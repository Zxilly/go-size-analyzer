package entity

import (
	"cmp"
	"fmt"
	"slices"
)

func fileBackedSize(s *Section) uint64 {
	if s.FileSize == 0 {
		return s.Size
	}
	return min(s.Size, s.FileSize)
}

type Store struct {
	Sections          map[string]*Section
	DataSectionsCache []AddrPos
	TextSectionsCache []AddrPos

	// sorted by Addr for binary search in FindSection
	sortedSections []*Section
}

func NewStore() *Store {
	return &Store{
		Sections:          make(map[string]*Section),
		DataSectionsCache: make([]AddrPos, 0),
		TextSectionsCache: make([]AddrPos, 0),
	}
}

// searchSorted finds the element in a sorted slice whose range [start, end) contains [addr, addr+size].
// getStart returns the range start for an element; contains checks the full containment.
// Returns the index of the matching element, or -1 if not found.
func searchSorted[T any](slice []T, addr uint64, getStart func(T) uint64, contains func(T, uint64, uint64) bool, size uint64) int {
	idx, _ := slices.BinarySearchFunc(slice, addr, func(elem T, target uint64) int {
		return cmp.Compare(getStart(elem), target)
	})
	// BinarySearchFunc returns the insertion point where getStart(elem) >= addr.
	// The containing element may be at idx-1 (starts before addr) or idx (exact match).
	for i := idx; i >= idx-1 && i >= 0; i-- {
		if i >= len(slice) {
			continue
		}
		if contains(slice[i], addr, size) {
			return i
		}
	}
	return -1
}

func (s *Store) FindSection(addr, size uint64) *Section {
	idx := searchSorted(s.sortedSections, addr,
		func(sect *Section) uint64 { return sect.Addr },
		func(sect *Section, a, sz uint64) bool { return sect.Addr <= a && a+sz <= sect.AddrEnd },
		size,
	)
	if idx < 0 {
		return nil
	}
	return s.sortedSections[idx]
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

func sortAddrPos(a []AddrPos) {
	slices.SortFunc(a, func(x, y AddrPos) int {
		return cmp.Compare(x.Addr, y.Addr)
	})
}

func (s *Store) BuildCache() {
	for _, section := range s.Sections {
		if section.Debug || section.OnlyInMemory {
			continue
		}

		switch section.ContentType {
		case SectionContentText:
			s.TextSectionsCache = append(s.TextSectionsCache, AddrPos{
				Addr: section.Addr,
				Size: fileBackedSize(section),
				Type: AddrTypeText,
			})
		case SectionContentData:
			s.DataSectionsCache = append(s.DataSectionsCache, AddrPos{
				Addr: section.Addr,
				Size: fileBackedSize(section),
				Type: AddrTypeData,
			})
		default:
			// ignore
		}
	}

	sortAddrPos(s.DataSectionsCache)
	sortAddrPos(s.TextSectionsCache)

	// build sorted section list for FindSection (exclude debug, only-in-memory and other)
	s.sortedSections = make([]*Section, 0, len(s.Sections))
	for _, section := range s.Sections {
		if section.Debug || section.OnlyInMemory || section.ContentType == SectionContentOther {
			continue
		}
		s.sortedSections = append(s.sortedSections, section)
	}
	slices.SortFunc(s.sortedSections, func(a, b *Section) int {
		return cmp.Compare(a.Addr, b.Addr)
	})
}

func (s *Store) IsData(addr, size uint64) bool {
	return searchSorted(s.DataSectionsCache, addr,
		func(ap AddrPos) uint64 { return ap.Addr },
		func(ap AddrPos, a, sz uint64) bool { return ap.Addr <= a && a+sz <= ap.Addr+ap.Size },
		size,
	) >= 0
}

func (s *Store) IsText(addr, size uint64) bool {
	return searchSorted(s.TextSectionsCache, addr,
		func(ap AddrPos) uint64 { return ap.Addr },
		func(ap AddrPos, a, sz uint64) bool { return ap.Addr <= a && a+sz <= ap.Addr+ap.Size },
		size,
	) >= 0
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
