package wrapper

import (
	"debug/dwarf"
	"debug/macho"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type MachoWrapper struct {
	file *macho.File
}

func (m *MachoWrapper) DWARF() (*dwarf.Data, error) {
	return m.file.DWARF()
}

func (m *MachoWrapper) LoadSymbols(marker func(name string, addr uint64, size uint64, typ entity.AddrType), goStrSymCb func(addr, size uint64)) error {
	if m.file.Symtab == nil || len(m.file.Symtab.Syms) == 0 {
		return ErrNoSymbolTable
	}

	const stabTypeMask = 0xe0

	syms := make([]macho.Symbol, 0)
	for _, s := range m.file.Symtab.Syms {
		if ignoreSymbols.Contains(s.Name) {
			continue
		}
		if s.Type&stabTypeMask != 0 {
			continue // skip stab debug info
		}

		syms = append(syms, s)
	}

	var addrs []uint64
	for _, s := range syms {
		addrs = append(addrs, s.Value)
	}
	slices.Sort(addrs)

	for _, s := range syms {
		i, ok := slices.BinarySearch(addrs, s.Value)
		if !ok {
			// maybe we met the last symbol, no way to get the CodeSize
			continue
		}
		size := addrs[i] - s.Value

		if s.Sect == 0 {
			continue // unknown
		}

		typ := entity.AddrTypeUnknown
		if int(s.Sect) > len(m.file.Sections) {
			continue // broken index
		}
		sect := m.file.Sections[s.Sect-1]

		switch sect.Seg {
		case "__DATA_CONST", "__DATA":
			typ = entity.AddrTypeData
		case "__TEXT":
			typ = entity.AddrTypeText
		}

		if machoSectionShouldIgnore(sect) {
			continue // bss section, skip
		}

		if s.Name == GoStringSymbol {
			goStrSymCb(s.Value, size)
			if marker == nil {
				return nil
			}
			continue
		}
		if marker != nil {
			marker(s.Name, s.Value, size, typ)
		}
	}

	return nil
}

func machoSectionType(s *macho.Section) entity.SectionContentType {
	switch {
	case s.Name == "__text":
		return entity.SectionContentText
	case strings.HasSuffix(s.Name, "bss") || strings.HasSuffix(s.Name, "data"):
		return entity.SectionContentData
	default:
		return entity.SectionContentOther
	}
}

func (m *MachoWrapper) LoadSections() *entity.Store {
	ret := entity.NewStore()

	for _, s := range m.file.Sections {
		if s.Size == 0 {
			continue
		}

		d := strings.HasPrefix(s.Name, "__debug_") || strings.HasPrefix(s.Name, "__zdebug_")

		if machoSectionShouldIgnore(s) {
			// seems like .bss section
			ret.Sections[s.Name] = &entity.Section{
				Name:         s.Name,
				Addr:         s.Addr,
				AddrEnd:      s.Addr + s.Size,
				OnlyInMemory: true,
				Debug:        d,
				ContentType:  machoSectionType(s),
			}
			continue
		}

		name := s.Name + " " + s.Seg

		if _, ok := ret.Sections[name]; ok {
			panic(fmt.Errorf("section %s already exists", name))
		}

		ret.Sections[name] = &entity.Section{
			Name:         name,
			Size:         s.Size,
			FileSize:     s.Size,
			Offset:       uint64(s.Offset),
			End:          uint64(s.Offset) + s.Size,
			Addr:         s.Addr,
			AddrEnd:      s.Addr + s.Size,
			OnlyInMemory: false,
			Debug:        d,
			ContentType:  machoSectionType(s),
		}
	}
	return ret
}

func machoSectionShouldIgnore(sect *macho.Section) bool {
	if sect.Seg == "__DATA" && (sect.Name == "__bss" || sect.Name == "__noptrbss") {
		return true
	}

	if sect.Offset == 0 {
		return true
	}

	const sZeroFill = 0x1
	const sGBZeroFill = 0xc

	if sect.Flags&sZeroFill != 0 {
		return true
	}

	if sect.Flags&sGBZeroFill != 0 {
		return true
	}

	return false
}

func (m *MachoWrapper) ReadAddr(addr, size uint64) ([]byte, error) {
	for _, load := range m.file.Loads {
		seg, ok := load.(*macho.Segment)
		if !ok {
			continue
		}
		if seg.Addr <= addr && addr <= seg.Addr+seg.Filesz-1 {
			if seg.Name == "__PAGEZERO" {
				continue
			}
			n := seg.Addr + seg.Filesz - addr
			if size > n {
				return nil, errors.New("size too large")
			}
			data := make([]byte, size)
			if _, err := seg.ReadAt(data, int64(addr-seg.Addr)); err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, ErrAddrNotFound
}

func (m *MachoWrapper) Text() (textStart uint64, text []byte, err error) {
	sect := m.file.Section("__text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return textStart, text, err
}

func (m *MachoWrapper) GoArch() string {
	switch m.file.Cpu {
	case macho.Cpu386:
		return "386"
	case macho.CpuAmd64:
		return "amd64"
	case macho.CpuArm:
		return "arm"
	case macho.CpuArm64:
		return "arm64"
	case macho.CpuPpc64:
		return "ppc64"
	}
	return ""
}
