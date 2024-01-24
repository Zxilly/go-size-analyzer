package pkg

import (
	"errors"
	"github.com/Zxilly/go-size-analyzer/pkg/tool"
	"github.com/goretk/gore"
	"strings"
)

type KnownInfo struct {
	Size       uint64
	BuildInfo  *gore.BuildInfo
	SectionMap *SectionMap
	Packages   *TypedPackages
	FoundAddr  *FoundAddr

	IsDynamicLink bool

	Version struct {
		Leq118 bool
		Meq120 bool
	}

	// not using right now
	GoStrSymbol struct {
		Start uint64
		Size  uint64
		Found bool
	}
}

var ErrPackageNotFound = errors.New("package not Found")

// MarkKnownPartWithPackageStr mark the part of the memory as known, should only be called after extractPackages
func (b *KnownInfo) MarkKnownPartWithPackageStr(start uint64, size uint64, pkg string, pass AddrParsePass, meta string) error {
	pkgPtr, ok := b.Packages.NameToPkg[pkg]
	if !ok {
		return errors.Join(ErrPackageNotFound, errors.New(pkg))
	}

	return b.FoundAddr.Insert(start, size, pkgPtr, pass, meta)
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
	if b.Version.Meq120 && (strings.HasPrefix(name, "go:") || strings.HasPrefix(name, "type:")) {
		return ""
	}

	// For go1.18 and below, the prefix are "type." and "go." instead.
	if b.Version.Leq118 && (strings.HasPrefix(name, "go.") || strings.HasPrefix(name, "type.")) {
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

func Collect(file *gore.GoFile) error {
	b := &KnownInfo{}

	b.FoundAddr = NewFoundAddr()

	b.SectionMap = extractSectionsFromGoFile(file)
	b.Size = tool.GetFileSize(file.GetFile())
	b.BuildInfo = file.BuildInfo

	if b.BuildInfo != nil && b.BuildInfo.Compiler != nil {
		b.Version.Leq118 = gore.GoVersionCompare(b.BuildInfo.Compiler.Name, "go1.18") <= 0
		b.Version.Meq120 = gore.GoVersionCompare(b.BuildInfo.Compiler.Name, "go1.20") >= 0
	} else {
		// if we can't get build info, we assume it's go1.20 plus
		b.Version.Meq120 = true
	}

	assertSectionsSize(b.SectionMap, b.Size)

	// this also increase the known Size of sections
	pkgs, err := extractPackages(file, b)
	if err != nil {
		return err
	}
	b.Packages = pkgs

	err = analysisSymbol(file, b)
	if err != nil {
		return err
	}

	err = TryExtractWithDisasm(file, b)
	if err != nil {
		return err
	}

	err = b.FoundAddr.AssertOverLap()
	if err != nil {
		return err
	}

	return nil
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
