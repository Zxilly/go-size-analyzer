package go_size_view

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"github.com/Zxilly/go-size-view/go-size-view/objfile"
	"github.com/goretk/gore"
	"log"
)

func extractSectionsFromGoFile(gofile *gore.GoFile) (sections *SectionMap) {
	switch f := gofile.GetParsedFile().(type) {
	case *pe.File:
		sections = extractSectionsFromPe(f)
	case *elf.File:
		sections = extractSectionsFromElf(f)
	case *macho.File:
		sections = extractSectionsFromMacho(f)
	}

	sections.SymTab = SymbolTable{Symbols: make(map[uint64]*Symbol)}

	obj, _ := objfile.Create(gofile.GetFile(), gofile.GetParsedFile())
	syms, err := obj.Symbols()
	if err != nil {
		log.Println("Warning: failed to get symbols. deep analysis will not be available")
		return
	}

	for _, sym := range syms {
		sections.SymTab.Symbols[sym.Addr] = &Symbol{Sym: sym, SizeCounted: false}
	}

	return
}

func assertSectionsSize(sections *SectionMap, size uint64) {
	sectionsSize := uint64(0)
	for _, section := range sections.Sections {
		sectionsSize += section.TotalSize
	}

	if sectionsSize > size {
		log.Fatalf("Error: sections size is bigger than file size. sections size: %d, file size: %d", sectionsSize, size)
	}
}

func extractSectionsFromPe(file *pe.File) (ret *SectionMap) {
	ret = &SectionMap{Sections: make(map[string]*Section)}

	getimageBase := func(file *pe.File) uint64 {
		if file.Machine == pe.IMAGE_FILE_MACHINE_I386 {
			optHdr := file.OptionalHeader.(*pe.OptionalHeader32)
			return uint64(optHdr.ImageBase)
		} else {
			optHdr := file.OptionalHeader.(*pe.OptionalHeader64)
			return optHdr.ImageBase
		}
	}

	imageBase := getimageBase(file)

	for _, section := range file.Sections {
		if section.Size == 0 {
			continue
		}

		ret.Sections[section.Name] = &Section{
			Name:      section.Name,
			TotalSize: uint64(section.Size),
			KnownSize: 0,
			Offset:    uint64(section.Offset),
			End:       uint64(section.Offset + section.Size),
			GoAddr:    imageBase + uint64(section.VirtualAddress),
			GoEnd:     imageBase + uint64(section.VirtualAddress+section.Size),
		}
	}
	return
}

func extractSectionsFromElf(file *elf.File) (ret *SectionMap) {
	ret = &SectionMap{Sections: make(map[string]*Section)}

	for _, section := range file.Sections {
		// not exist in binary
		if section.Type == elf.SHT_NULL || section.Size == 0 {
			continue
		}

		if section.Type == elf.SHT_NOBITS {
			// seems like .bss section
			ret.Sections[section.Name] = &Section{
				Name:   section.Name,
				GoAddr: section.Addr,
				GoEnd:  section.Addr + section.Size,
			}
			continue
		}

		ret.Sections[section.Name] = &Section{
			Name:      section.Name,
			TotalSize: section.FileSize,
			KnownSize: 0,
			Offset:    section.Offset,
			End:       section.Offset + section.FileSize,
			GoAddr:    section.Addr,
			GoEnd:     section.Addr + section.Size,
		}
	}

	return
}

func extractSectionsFromMacho(file *macho.File) (ret *SectionMap) {
	ret = &SectionMap{
		Sections: map[string]*Section{},
		SymTab: SymbolTable{
			Symbols: make(map[uint64]*Symbol),
		},
	}

	for _, section := range file.Sections {
		if section.Size == 0 {
			continue
		}

		ret.Sections[section.Name] = &Section{
			Name:      section.Name,
			TotalSize: section.Size,
			KnownSize: 0,
			Offset:    uint64(section.Offset),
			End:       uint64(section.Offset) + section.Size,
			GoAddr:    section.Addr,
			GoEnd:     section.Addr + section.Size,
		}
	}

	return
}
