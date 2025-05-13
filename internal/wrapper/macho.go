package wrapper

import (
	"bytes"
	"compress/zlib"
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/types"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type MachoWrapper struct {
	file           *macho.File
	chainedFixedUp bool
}

var _ RawFileWrapper = (*MachoWrapper)(nil)

func NewMachoWrapper(f *macho.File) *MachoWrapper {
	return &MachoWrapper{
		file:           f,
		chainedFixedUp: f.HasDyldChainedFixups(),
	}
}

func (m *MachoWrapper) SlidePointer(addr uint64) uint64 {
	if !m.chainedFixedUp {
		return addr
	}
	return m.file.SlidePointer(addr) + m.file.GetBaseAddress()
}

// DWARF a copy of go-macho's DWARF function
func (m *MachoWrapper) DWARF() (*dwarf.Data, error) {
	dwarfSuffix := func(s *types.Section) string {
		switch {
		case strings.HasPrefix(s.Name, "__debug_"):
			return s.Name[8:]
		case strings.HasPrefix(s.Name, "__zdebug_"):
			return s.Name[9:]
		default:
			return ""
		}
	}
	sectionData := func(s *types.Section) ([]byte, error) {
		b, err := s.Data()
		if err != nil && uint64(len(b)) < s.Size {
			return nil, err
		}

		if len(b) >= 12 && string(b[:4]) == "ZLIB" {
			dlen := binary.BigEndian.Uint64(b[4:12])
			dbuf := make([]byte, dlen)
			r, err := zlib.NewReader(bytes.NewBuffer(b[12:]))
			if err != nil {
				return nil, err
			}
			if _, err := io.ReadFull(r, dbuf); err != nil {
				return nil, err
			}
			if err := r.Close(); err != nil {
				return nil, err
			}
			b = dbuf
		}
		return b, nil
	}

	// There are many other DWARF sections, but these
	// are the ones the debug/dwarf package uses.
	// Don't bother loading others.
	dat := map[string][]byte{"abbrev": nil, "info": nil, "str": nil, "line": nil, "ranges": nil}
	for _, s := range m.file.Sections {
		suffix := dwarfSuffix(s)
		if suffix == "" {
			continue
		}
		if _, ok := dat[suffix]; !ok {
			continue
		}
		b, err := sectionData(s)
		if err != nil {
			return nil, err
		}
		dat[suffix] = b
	}

	d, err := dwarf.New(dat["abbrev"], nil, nil, dat["info"], dat["line"], nil, dat["ranges"], dat["str"])
	if err != nil {
		return nil, err
	}

	// Look for DWARF4 .debug_types sections and DWARF5 sections.
	for i, s := range m.file.Sections {
		suffix := dwarfSuffix(s)
		if suffix == "" {
			continue
		}
		if _, ok := dat[suffix]; ok {
			// Already handled.
			continue
		}

		b, err := sectionData(s)
		if err != nil {
			return nil, err
		}

		if suffix == "types" {
			err = d.AddTypes(fmt.Sprintf("types-%d", i), b)
		} else {
			err = d.AddSection(".debug_"+suffix, b)
		}
		if err != nil {
			return nil, err
		}
	}

	return d, nil
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

func machoSectionType(s *types.Section) entity.SectionContentType {
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

func machoSectionShouldIgnore(sect *types.Section) bool {
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
	addr = m.file.SlidePointer(addr)
	off, err := m.file.GetOffset(addr)
	if err != nil {
		return nil, ErrAddrNotFound
	}
	b := make([]byte, size)
	if size == 0 {
		return b, nil
	}
	_, err = m.file.ReadAt(b, int64(off))
	return b, err
}

func (m *MachoWrapper) Text() (textStart uint64, text []byte, err error) {
	var sect *types.Section
	for _, s := range m.file.Sections {
		if s.Name == "__text" {
			sect = s
			break
		}
	}
	if sect == nil {
		return 0, nil, errors.New("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return textStart, text, err
}

func (m *MachoWrapper) GoArch() string {
	switch m.file.CPU {
	case types.CPUI386:
		return "386"
	case types.CPUAmd64:
		return "amd64"
	case types.CPUArm:
		return "arm"
	case types.CPUArm64:
		return "arm64"
	case types.CPUPpc64:
		return "ppc64"
	}
	return ""
}
