package pkg

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"github.com/Zxilly/go-size-analyzer/pkg/tool"
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
	default:
		panic("This should not happened :(")
	}
	return
}

func assertSectionsSize(sections *SectionMap, size uint64) {
	sectionsSize := uint64(0)
	for _, section := range sections.Sections {
		sectionsSize += section.TotalSize
	}

	if sectionsSize > size {
		log.Fatalf("Error: sections Size is bigger than file Size. sections Size: %d, file Size: %d", sectionsSize, size)
	}
}

func extractSectionsFromPe(file *pe.File) (ret *SectionMap) {
	ret = &SectionMap{Sections: make(map[string]*Section)}

	imageBase := tool.GetImageBase(file)

	for _, section := range file.Sections {
		ret.Sections[section.Name] = &Section{
			Name:         section.Name,
			TotalSize:    uint64(section.Size),
			Offset:       uint64(section.Offset),
			End:          uint64(section.Offset + section.Size),
			Addr:         imageBase + uint64(section.VirtualAddress),
			AddrEnd:      imageBase + uint64(section.VirtualAddress+section.VirtualSize),
			OnlyInMemory: false, // pe file not set only in memory section
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
				Name:         section.Name,
				Addr:         section.Addr,
				AddrEnd:      section.Addr + section.Size,
				OnlyInMemory: true,
			}
			continue
		}

		ret.Sections[section.Name] = &Section{
			Name:         section.Name,
			TotalSize:    section.FileSize,
			Offset:       section.Offset,
			End:          section.Offset + section.FileSize,
			Addr:         section.Addr,
			AddrEnd:      section.Addr + section.Size,
			OnlyInMemory: false,
		}
	}

	return
}

func extractSectionsFromMacho(file *macho.File) (ret *SectionMap) {
	ret = &SectionMap{Sections: map[string]*Section{}}

	for _, section := range file.Sections {
		if section.Size == 0 {
			continue
		}

		if section.Offset == 0 {
			// seems like .bss section
			ret.Sections[section.Name] = &Section{
				Name:         section.Name,
				Addr:         section.Addr,
				AddrEnd:      section.Addr + section.Size,
				OnlyInMemory: true,
			}
			continue
		}

		ret.Sections[section.Name] = &Section{
			Name:         section.Name,
			TotalSize:    section.Size,
			Offset:       uint64(section.Offset),
			End:          uint64(section.Offset) + section.Size,
			Addr:         section.Addr,
			AddrEnd:      section.Addr + section.Size,
			OnlyInMemory: false,
		}
	}

	return
}

type SectionMap struct {
	Sections map[string]*Section
}

func (s *SectionMap) GetSectionName(addr uint64) string {
	for _, section := range s.Sections {
		if addr >= section.Addr && addr < section.AddrEnd {
			return section.Name
		}
	}
	return ""
}

func (s *SectionMap) GetSection(addr, size uint64) *Section {
	for _, section := range s.Sections {
		if addr >= section.Addr && addr < section.AddrEnd && addr+size <= section.AddrEnd {
			return section
		}
	}
	return nil
}
