package section

import (
	"fmt"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type Store struct {
	Sections map[string]*entity.Section
}

func (s *Store) FindSection(addr, size uint64) *entity.Section {
	for _, section := range s.Sections {
		if section.Debug {
			// we can't find things in debug sections
			continue
		}

		if section.Addr <= addr && addr+size <= section.AddrEnd {
			return section
		}
	}
	return nil
}

func (s *Store) AssertSize(size uint64) error {
	sectionsSize := uint64(0)
	for _, section := range s.Sections {
		if section.OnlyInMemory {
			continue
		}
		sectionsSize += section.FileSize
	}

	if sectionsSize > size {
		return fmt.Errorf("section size %d > file size %d", sectionsSize, size)
	}

	return nil
}
