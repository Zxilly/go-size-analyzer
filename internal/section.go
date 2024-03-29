package internal

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type SectionMap struct {
	Sections map[string]*entity.Section
}

func (s *SectionMap) GetSectionName(addr uint64) string {
	for _, section := range s.Sections {
		if addr >= section.Addr && addr < section.AddrEnd {
			return section.Name
		}
	}
	return ""
}

func (s *SectionMap) GetSection(addr, size uint64) *entity.Section {
	for _, section := range s.Sections {
		if addr >= section.Addr && addr < section.AddrEnd && addr+size <= section.AddrEnd {
			return section
		}
	}
	return nil
}

func (s *SectionMap) AssertSize(size uint64) error {
	sectionsSize := uint64(0)
	for _, section := range s.Sections {
		if section.OnlyInMemory {
			continue
		}
		sectionsSize += section.Size
	}

	if sectionsSize > size {
		return fmt.Errorf("section size %d > file size %d", sectionsSize, size)
	}

	return nil
}
