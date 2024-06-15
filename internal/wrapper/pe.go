package wrapper

import (
	"debug/dwarf"
	"debug/pe"
	"fmt"
	"slices"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type PeWrapper struct {
	file      *pe.File
	imageBase uint64
}

func (p *PeWrapper) ImageBase() uint64 {
	return p.imageBase
}

func (p *PeWrapper) DWARF() (*dwarf.Data, error) {
	return p.file.DWARF()
}

func (*PeWrapper) PclntabSections() []string {
	return []string{".rdata"} // FIXME: get real position from gore, can be .text
}

func (p *PeWrapper) LoadSymbols(marker func(name string, addr uint64, size uint64, typ entity.AddrType) error) error {
	if len(p.file.Symbols) == 0 {
		return ErrNoSymbolTable
	}

	const (
		nUndef = 0
		nAbs   = -1
		nDebug = -2
	)

	type sym struct {
		Name string
		Addr uint64
		Size uint64
		Typ  entity.AddrType
	}

	peSyms := make([]*pe.Symbol, 0)
	for _, s := range p.file.Symbols {
		if s.SectionNumber == nDebug || s.SectionNumber == nAbs || s.SectionNumber == nUndef {
			continue // not addr, skip
		}
		if s.SectionNumber < 0 || len(p.file.Sections) < int(s.SectionNumber) {
			return fmt.Errorf("invalid section number in symbol table")
		}
		if ignoreSymbols.Contains(s.Name) {
			continue
		}
		peSyms = append(peSyms, s)
	}

	syms := make([]sym, 0)
	addrs := make([]uint64, 0)

	for _, s := range peSyms {
		const (
			text = 0x20
			data = 0x40
		)

		sect := p.file.Sections[s.SectionNumber-1]
		ch := sect.Characteristics

		a := uint64(s.Value) + p.imageBase + uint64(sect.VirtualAddress)

		var typ entity.AddrType
		switch {
		case ch&text != 0:
			typ = entity.AddrTypeText
		case ch&data != 0:
			typ = entity.AddrTypeData
		default:
			continue // not text/data, skip
		}

		syms = append(syms, sym{
			Name: s.Name,
			Addr: a,
			Typ:  typ,
			Size: 0, // will be filled later
		})

		addrs = append(addrs, a)
	}

	slices.Sort(addrs)

	for _, s := range syms {
		i, ok := slices.BinarySearch(addrs, s.Addr)
		if !ok {
			// Maybe we met the last symbol, skip it, no way to get the CodeSize
			continue
		}
		size := addrs[i] - s.Addr

		s.Size = size

		err := marker(s.Name, s.Addr, size, s.Typ)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PeWrapper) LoadSections() map[string]*entity.Section {
	ret := make(map[string]*entity.Section)
	for _, section := range p.file.Sections {
		d := strings.HasPrefix(section.Name, ".debug_") || strings.HasPrefix(section.Name, ".zdebug_")

		if _, ok := ret[section.Name]; ok {
			panic(fmt.Sprintf("section %s already exists", section.Name))
		}

		ret[section.Name] = &entity.Section{
			Name:         section.Name,
			Size:         uint64(section.VirtualSize),
			FileSize:     uint64(section.Size),
			Offset:       uint64(section.Offset),
			End:          uint64(section.Offset + section.Size),
			Addr:         p.imageBase + uint64(section.VirtualAddress),
			AddrEnd:      p.imageBase + uint64(section.VirtualAddress+section.VirtualSize),
			OnlyInMemory: false, // pe file didn't have an only-in-memory section
			Debug:        d,
		}
	}
	return ret
}

func (p *PeWrapper) ReadAddr(addr, size uint64) ([]byte, error) {
	pf := p.file
	for _, sect := range pf.Sections {
		if uint64(sect.VirtualAddress) <= addr && addr+size <= uint64(sect.VirtualAddress+sect.VirtualSize) {
			data := make([]byte, size)
			if _, err := sect.ReadAt(data, int64(addr-uint64(sect.VirtualAddress))); err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, ErrAddrNotFound
}

func (p *PeWrapper) Text() (textStart uint64, text []byte, err error) {
	sect := p.file.Section(".text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = p.imageBase + uint64(sect.VirtualAddress)
	text, err = sect.Data()
	return textStart, text, err
}

func (p *PeWrapper) GoArch() string {
	switch p.file.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		return "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "amd64"
	case pe.IMAGE_FILE_MACHINE_ARMNT:
		return "arm"
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return "arm64"
	default:
		return ""
	}
}
