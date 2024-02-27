package pkg

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"errors"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/pkg/tool"
	"slices"
)

var ErrNoSymbolTable = errors.New("no symbol table found")

func analyzePeSymbol(f *pe.File, b *KnownInfo) error {
	if len(f.Symbols) == 0 {
		return ErrNoSymbolTable
	}

	imageBase := tool.GetImageBase(f)

	const (
		NUndef = 0
		NAbs   = -1
		NDebug = -2
	)

	type sym struct {
		Name string
		Addr uint64
		Size uint64
		Typ  AddrType
	}

	peSyms := make([]*pe.Symbol, 0)
	for _, s := range f.Symbols {
		if s.SectionNumber == NDebug || s.SectionNumber == NAbs || s.SectionNumber == NUndef {
			continue // not addr, skip
		}
		if s.SectionNumber < 0 || len(f.Sections) < int(s.SectionNumber) {
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

		sect := f.Sections[s.SectionNumber-1]
		ch := sect.Characteristics

		addr := uint64(s.Value) + imageBase + uint64(sect.VirtualAddress)

		typ := AddrTypeUnknown
		switch {
		case ch&text != 0:
			typ = AddrTypeText
		case ch&data != 0:
			typ = AddrTypeData
		default:
			continue // not text/data, skip
		}

		syms = append(syms, sym{
			Name: s.Name,
			Addr: addr,
			Typ:  typ,
			Size: 0, // will be filled later
		})

		addrs = append(addrs, addr)
	}

	slices.Sort(addrs)

	for _, s := range syms {
		i, ok := slices.BinarySearch(addrs, s.Addr)
		if !ok {
			// Maybe we met the last symbol, skip it, no way to get the Size
			continue
		}
		size := addrs[i] - s.Addr

		s.Size = size

		err := b.MarkSymbol(s.Name, s.Addr, size, s.Typ)
		if err != nil {
			return err
		}
	}

	return nil
}

func analyzeElfSymbol(f *elf.File, b *KnownInfo) error {
	symbols, err := f.Symbols()
	if err != nil {
		if errors.Is(err, elf.ErrNoSymbols) {
			return ErrNoSymbolTable
		}
		return err
	}

	keep := make([]elf.Symbol, 0)
	for _, s := range symbols {
		if ignoreSymbols.Contains(s.Name) {
			continue
		}
		keep = append(keep, s)
	}
	symbols = keep

	for _, s := range symbols {
		switch s.Section {
		case elf.SHN_UNDEF, elf.SHN_ABS, elf.SHN_COMMON:
			continue // not addr, skip
		}

		if s.Size == 0 {
			// nothing to do
			continue
		}

		i := int(s.Section)
		if i < 0 || i >= len(f.Sections) {
			// just ignore, exmaple: we met go.go
			continue
		}
		sect := f.Sections[i]
		typ := AddrTypeUnknown
		switch sect.Flags & (elf.SHF_WRITE | elf.SHF_ALLOC | elf.SHF_EXECINSTR) {
		case elf.SHF_ALLOC | elf.SHF_EXECINSTR:
			typ = AddrTypeText
		case elf.SHF_ALLOC:
			typ = AddrTypeData
		default:
			continue // wtf?
		}

		err = b.MarkSymbol(s.Name, s.Value, s.Size, typ)
		if err != nil {
			return err
		}
	}

	return nil
}

func analyzeMachoSymbol(f *macho.File, b *KnownInfo) error {
	if f.Symtab == nil {
		return ErrNoSymbolTable
	}

	const stabTypeMask = 0xe0

	syms := make([]macho.Symbol, 0)
	for _, s := range f.Symtab.Syms {
		if ignoreSymbols.Contains(s.Name) {
			continue
		}
		if s.Type&stabTypeMask != 0 {
			continue // skip stab debug info
		}
	}

	var addrs []uint64
	for _, s := range syms {
		addrs = append(addrs, s.Value)
	}
	slices.Sort(addrs)

	for _, s := range syms {
		i, ok := slices.BinarySearch(addrs, s.Value)
		if !ok {
			// maybe we met the last symbol, no way to get the Size
			continue
		}
		size := addrs[i] - s.Value

		if s.Sect == 0 {
			continue // unknown
		}

		typ := AddrTypeUnknown
		if int(s.Sect) <= len(f.Sections) {
			sect := f.Sections[s.Sect-1]

			switch sect.Seg {
			case "__DATA_CONST", "__DATA":
				typ = AddrTypeData
			case "__TEXT":
				typ = AddrTypeText
			}

			if sect.Seg == "__DATA" && (sect.Name == "__bss" || sect.Name == "__noptrbss") {
				continue // bss section, skip
			}
		} else {
			continue // broken index
		}

		err := b.MarkSymbol(s.Name, s.Value, size, typ)
		if err != nil {
			return err
		}
	}

	return nil
}
