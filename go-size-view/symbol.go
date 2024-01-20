package go_size_view

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"errors"
	"fmt"
	"github.com/goretk/gore"
	"log"
	"slices"
)

func collectSizeFromSymbol(file *gore.GoFile, b *KnownInfo) error {
	switch f := file.GetParsedFile().(type) {
	case *pe.File:
		return collectSizeFromPeSymbol(f, b)
	case *elf.File:
		return collectSizeFromElfSymbol(f, b)
	case *macho.File:
		return collectSizeFromMachoSymbol(f, b)
	default:
		panic("This should not happened :(")
	}
}

func collectSizeFromPeSymbol(f *pe.File, b *KnownInfo) error {
	imageBase := getimageBase(f)

	const (
		NUndef = 0
		NAbs   = -1
		NDebug = -2
	)

	type sym struct {
		Name string
		Addr uint64
		Size uint64
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
		if ch&text == 0 && ch&data == 0 {
			continue // not text/data, skip
		}

		syms = append(syms, sym{
			Name: s.Name,
			Addr: uint64(s.Value),
			Size: 0, // will be filled later
		})

		addrs = append(addrs, uint64(s.Value)+imageBase+uint64(sect.VirtualAddress))
	}

	slices.Sort(addrs)

	for _, s := range syms {
		i, ok := slices.BinarySearch(addrs, s.Addr)
		if !ok {
			// Maybe we met the last symbol, skip it, no way to get the size
			continue
		}
		size := addrs[i] - s.Addr

		pkgName := b.ExtractPackageFromSymbol(s.Name)
		if pkgName == "" {
			continue // skip compiler-generated symbols
		}

		err := b.MarkKnownPartWithPackage(s.Addr, size, pkgName)
		if err != nil {
			if errors.Is(err, ErrPackageNotFound) {
				continue // some symbol like complex symbol or cgo symbol, can't find
			}
			return err
		}
	}
	return nil
}

func collectSizeFromElfSymbol(f *elf.File, b *KnownInfo) error {
	symbols, err := f.Symbols()
	if err != nil {
		if errors.Is(err, elf.ErrNoSymbols) {
			log.Println("Warning: no symbol table found")
			return nil // keep going without symbol table
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
		if s.Section == elf.SHN_UNDEF || s.Section == elf.SHN_COMMON {
			continue // skip undefined/bss symbols
		}

		i := int(s.Section)
		if i < 0 || i >= len(f.Sections) {
			return fmt.Errorf("invalid section number in symbol table")
		}
		sect := f.Sections[i]
		switch sect.Flags & (elf.SHF_WRITE | elf.SHF_ALLOC | elf.SHF_EXECINSTR) {
		case elf.SHF_ALLOC | elf.SHF_EXECINSTR:
		case elf.SHF_ALLOC:
		case elf.SHF_ALLOC | elf.SHF_WRITE:
		default:
			continue // not text/data, skip
		}

		pkgName := b.ExtractPackageFromSymbol(s.Name)
		if pkgName == "" {
			continue // skip compiler-generated symbols
		}

		err = b.MarkKnownPartWithPackage(s.Value, s.Size, pkgName)
		if err != nil {
			if errors.Is(err, ErrPackageNotFound) {
				continue // some symbol like complex symbol or cgo symbol, can't find
			}
			return err
		}
	}
	return nil
}

func collectSizeFromMachoSymbol(f *macho.File, b *KnownInfo) error {
	if f.Symtab == nil {
		log.Println("Warning: no symbol table found")
		return nil // keep going without symbol table
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
			// maybe we met the last symbol, no way to get the size
			continue
		}
		size := addrs[i] - s.Value

		if s.Sect == 0 {
			continue // unknown
		}

		if int(s.Sect) <= len(f.Sections) {
			sect := f.Sections[s.Sect-1]
			if sect.Seg != "__TEXT" && sect.Seg != "__DATA" && sect.Seg != "__DATA_CONST" {
				continue // not text/data, skip
			}

			if sect.Seg == "__DATA" && (sect.Name == "__bss" || sect.Name == "__noptrbss") {
				continue // bss section, skip
			}
		} else {
			continue // broken index
		}

		pkgName := b.ExtractPackageFromSymbol(s.Name)
		if pkgName == "" {
			continue // skip compiler-generated symbols
		}

		err := b.MarkKnownPartWithPackage(s.Value, size, pkgName)
		if err != nil {
			if errors.Is(err, ErrPackageNotFound) {
				continue // some symbol like complex symbol or cgo symbol, can't find
			}
			return err
		}
	}

	return nil
}
