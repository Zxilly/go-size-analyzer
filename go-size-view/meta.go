package go_size_view

import (
	"fmt"
	"github.com/goretk/gore"
	"strings"
)

type KnownInfo struct {
	Size       uint64
	BuildInfo  *gore.BuildInfo
	SectionMap *SectionMap
	Packages   *TypedPackages
	FoundAddr  *FoundAddr

	version struct {
		leq118 bool
		meq120 bool
	}
}

// MarkKnownPartWithPackage mark the part of the memory as known, should only be called after extractPackages
func (b *KnownInfo) MarkKnownPartWithPackage(start uint64, size uint64, pkg string) error {
	b.FoundAddr.Insert(start, size)
	pkgPtr, ok := b.Packages.nameToPkg[pkg]
	if !ok {
		return fmt.Errorf("package %s not found", pkg)
	}
	sectionName := b.SectionMap.GetSectionName(start)
	if sectionName == "" {
		return fmt.Errorf("section not found for addr %#x", start)
	}
	pkgPtr.Sections[sectionName] += size
	return nil
}

// ExtractPackageFromSymbol copied from debug/gosym/symtab.go
func (b *KnownInfo) ExtractPackageFromSymbol(s string) string {
	nameWithoutInst := func(name string) string {
		start := strings.Index(name, "[")
		if start < 0 {
			return name
		}
		end := strings.LastIndex(name, "]")
		if end < 0 {
			// Malformed name, should contain closing bracket too.
			return name
		}
		return name[0:start] + name[end+1:]
	}

	name := nameWithoutInst(s)

	// Since go1.20, a prefix of "type:" and "go:" is a compiler-generated symbol,
	// they do not belong to any package.
	//
	// See cmd/compile/internal/base/link.go:ReservedImports variable.
	if b.version.meq120 && (strings.HasPrefix(name, "go:") || strings.HasPrefix(name, "type:")) {
		return ""
	}

	// For go1.18 and below, the prefix are "type." and "go." instead.
	if b.version.leq118 && (strings.HasPrefix(name, "go.") || strings.HasPrefix(name, "type.")) {
		return ""
	}

	pathend := strings.LastIndex(name, "/")
	if pathend < 0 {
		pathend = 0
	}

	if i := strings.Index(name[pathend:], "."); i != -1 {
		return name[:pathend+i]
	}
	return ""
}

func (b *KnownInfo) GetPaddingSize() uint64 {
	var sectionSize uint64 = 0
	for _, section := range b.SectionMap.Sections {
		sectionSize += section.TotalSize
	}
	return b.Size - sectionSize
}

func (b *KnownInfo) Collect(file *gore.GoFile) error {
	b.FoundAddr = NewFoundAddr()

	b.SectionMap = extractSectionsFromGoFile(file)
	b.Size = getFileSize(file.GetFile())
	b.BuildInfo = file.BuildInfo

	b.version.leq118 = gore.GoVersionCompare(b.BuildInfo.Compiler.Name, "go1.18") <= 0
	b.version.meq120 = gore.GoVersionCompare(b.BuildInfo.Compiler.Name, "go1.20") >= 0

	assertSectionsSize(b.SectionMap, b.Size)

	// this also increase the known size of sections
	pkgs, err := extractPackages(file, b)
	if err != nil {
		return err
	}
	b.Packages = pkgs

	collectSizeFromSymbol(file, b)

	return nil
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

type Section struct {
	Name      string
	TotalSize uint64

	Offset uint64
	End    uint64

	Addr    uint64
	AddrEnd uint64

	OnlyInMemory bool
}

type File struct {
	Path      string
	Functions []*gore.Function
}

func (f *File) GetSize() uint64 {
	var size uint64 = 0
	for _, fn := range f.Functions {
		size += fn.End - fn.Offset
	}
	return size
}
