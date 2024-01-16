package go_size_view

import (
	"errors"
	"github.com/Zxilly/go-size-view/go-size-view/objfile"
	"github.com/goretk/gore"
)

type Bin struct {
	Size       uint64
	BuildInfo  *gore.BuildInfo
	SectionMap *SectionMap
	Packages   *TypedPackages
}

func (b *Bin) GetMetaSize() uint64 {
	var sectionSize uint64 = 0
	for _, section := range b.SectionMap.Sections {
		sectionSize += section.TotalSize
	}
	return b.Size - sectionSize
}

type Symbol struct {
	objfile.Sym
	SizeCounted bool
}

type SymbolTable struct {
	Symbols map[uint64]*Symbol // addr -> symbol
}

type SectionMap struct {
	Sections map[string]*Section
	SymTab   SymbolTable
}

func (s *SectionMap) IncreaseKnown(start uint64, end uint64) error {
	for _, section := range s.Sections {
		if start >= section.GoAddr && end <= section.GoEnd {
			if section.TotalSize == 0 {
				// just ignore, can be .bss or .noptrbss
				return nil
			}

			section.KnownSize += end - start
			if section.KnownSize > section.TotalSize {
				return errors.New("known size is bigger than total size")
			}

			return nil
		}
	}
	return errors.New("no section found")
}

type Section struct {
	Name      string
	TotalSize uint64
	KnownSize uint64 // has been calculated in the modules

	Offset uint64
	End    uint64

	GoAddr uint64
	GoEnd  uint64
}

type TypedPackages struct {
	Self      []*Packages
	Std       []*Packages
	Vendor    []*Packages
	Generated []*Packages
	Unknown   []*Packages
}

type Packages struct {
	Name     string
	Size     uint64
	Files    []*File
	Sections map[string]uint64
	grPkg    *gore.Package
}

type File struct {
	Size      uint64
	Path      string
	Functions []*gore.Function
}
