package entity

import (
	"fmt"
)

type Store struct {
	Sections          map[string]*Section
	DataSectionsCache []AddrPos
	TextSectionsCache []AddrPos
}

func NewStore() *Store {
	return &Store{
		Sections:          make(map[string]*Section),
		DataSectionsCache: make([]AddrPos, 0),
		TextSectionsCache: make([]AddrPos, 0),
	}
}

func (s *Store) FindSection(addr, size uint64) *Section {
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

func (s *Store) BuildCache() {
	for _, section := range s.Sections {
		if section.Debug {
			continue
		}

		switch section.ContentType {
		case SectionContentText:
			s.TextSectionsCache = append(s.TextSectionsCache, AddrPos{
				Addr: section.Addr,
				Size: section.Size,
				Type: AddrTypeText,
			})
		case SectionContentData:
			s.DataSectionsCache = append(s.DataSectionsCache, AddrPos{
				Addr: section.Addr,
				Size: section.Size,
				Type: AddrTypeData,
			})
		default:
			// ignore
		}
	}
}

func (s *Store) IsData(addr, size uint64) bool {
	for _, sect := range s.DataSectionsCache {
		if sect.Addr <= addr && addr+size <= sect.Addr+sect.Size {
			return true
		}
	}
	return false
}

func (s *Store) IsText(addr, size uint64) bool {
	for _, sect := range s.TextSectionsCache {
		if sect.Addr <= addr && addr+size <= sect.Addr+sect.Size {
			return true
		}
	}
	return false
}

func (s *Store) IsType(addr, size uint64, t AddrType) bool {
	switch t {
	case AddrTypeData:
		return s.IsData(addr, size)
	case AddrTypeText:
		return s.IsText(addr, size)
	default:
		return true
	}
}
