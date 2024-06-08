package wrapper

import (
	"debug/dwarf"
	"debug/macho"
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

func (*MachoWrapper) PclntabSections() []string {
	return []string{"__gopclntab __TEXT", "__gopclntab __DATA_CONST"}
}

func (m *MachoWrapper) LoadSymbols(marker func(name string, addr uint64, size uint64, typ entity.AddrType) error) error {
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

		if sect.Seg == "__DATA" && (sect.Name == "__bss" || sect.Name == "__noptrbss") {
			continue // bss section, skip
		}

		err := marker(s.Name, s.Value, size, typ)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MachoWrapper) LoadSections() map[string]*entity.Section {
	ret := make(map[string]*entity.Section)
	for _, section := range m.file.Sections {
		if section.Size == 0 {
			continue
		}

		d := strings.HasPrefix(section.Name, "__debug_") || strings.HasPrefix(section.Name, "__zdebug_")

		if section.Offset == 0 {
			// seems like .bss section
			ret[section.Name] = &entity.Section{
				Name:         section.Name,
				Addr:         section.Addr,
				AddrEnd:      section.Addr + section.Size,
				OnlyInMemory: true,
				Debug:        d,
			}
			continue
		}

		name := section.Name + " " + section.Seg

		if _, ok := ret[name]; ok {
			panic(fmt.Sprintf("section %s already exists", name))
		}

		ret[name] = &entity.Section{
			Name:         name,
			Size:         section.Size,
			FileSize:     section.Size,
			Offset:       uint64(section.Offset),
			End:          uint64(section.Offset) + section.Size,
			Addr:         section.Addr,
			AddrEnd:      section.Addr + section.Size,
			OnlyInMemory: false,
			Debug:        d,
		}
	}
	return ret
}

func (m *MachoWrapper) ReadAddr(addr, size uint64) ([]byte, error) {
	mf := m.file
	for _, sect := range mf.Sections {
		if sect.Addr <= addr && addr+size <= sect.Addr+sect.Size {
			data := make([]byte, size)
			if _, err := sect.ReadAt(data, int64(addr-sect.Addr)); err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("address not found")
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
