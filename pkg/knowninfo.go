package pkg

import (
	"errors"
	"github.com/Zxilly/go-size-analyzer/pkg/tool"
	"github.com/goretk/gore"
	"log"
	"strings"
)

type KnownInfo struct {
	Size       uint64
	BuildInfo  *gore.BuildInfo
	SectionMap *SectionMap
	Packages   *TypedPackages
	FoundAddr  *FoundAddr

	IsDynamicLink bool

	VersionFlag struct {
		Leq118 bool
		Meq120 bool
	}
}

// ExtractPackageFromSymbol copied from debug/gosym/symtab.go
func (k *KnownInfo) ExtractPackageFromSymbol(s string) string {
	nameWithoutInst := func(name string) string {
		start := strings.Index(name, "[")
		if start < 0 {
			return name
		}
		end := strings.LastIndex(name, "]")
		if end < 0 {
			// Malformed name should contain closing bracket too.
			return name
		}
		return name[0:start] + name[end+1:]
	}

	name := nameWithoutInst(s)

	// Since go1.20, a prefix of "type:" and "go:" is a compiler-generated symbol,
	// they do not belong to any package.
	//
	// See cmd/compile/internal/base/link.go: ReservedImports variable.
	if k.VersionFlag.Meq120 && (strings.HasPrefix(name, "go:") || strings.HasPrefix(name, "type:")) {
		return ""
	}

	// For go1.18 and below, the prefix is "type." and "go." instead.
	if k.VersionFlag.Leq118 && (strings.HasPrefix(name, "go.") || strings.HasPrefix(name, "type.")) {
		return ""
	}

	pathEnd := strings.LastIndex(name, "/")
	if pathEnd < 0 {
		pathEnd = 0
	}

	if i := strings.Index(name[pathEnd:], "."); i != -1 {
		return name[:pathEnd+i]
	}
	return ""
}

func (k *KnownInfo) GetPaddingSize() uint64 {
	var sectionSize uint64 = 0
	for _, section := range k.SectionMap.Sections {
		sectionSize += section.Size
	}
	return k.Size - sectionSize
}

func Collect(file *gore.GoFile) (*KnownInfo, error) {
	b := &KnownInfo{}

	b.FoundAddr = NewFoundAddr()

	b.SectionMap = loadSectionMap(file)
	b.Size = tool.GetFileSize(file.GetFile())
	b.BuildInfo = file.BuildInfo

	if b.BuildInfo != nil && b.BuildInfo.Compiler != nil {
		b.VersionFlag.Leq118 = gore.GoVersionCompare(b.BuildInfo.Compiler.Name, "go1.18") <= 0
		b.VersionFlag.Meq120 = gore.GoVersionCompare(b.BuildInfo.Compiler.Name, "go1.20") >= 0
	} else {
		// if we can't get build info, we assume it's go1.20 plus
		b.VersionFlag.Meq120 = true
	}

	err := b.SectionMap.AssertSize(b.Size)
	if err != nil {
		return nil, err
	}

	pkgs, err := b.loadPackages(file)
	if err != nil {
		return nil, err
	}
	b.Packages = pkgs

	err = b.analyzeSymbol(file)
	if err != nil {
		if errors.Is(err, ErrNoSymbolTable) {
			log.Println("Warning: no symbol table found, this can lead to inaccurate results")
		} else {
			return nil, err
		}
	}

	err = b.tryDisasm(file)
	if err != nil {
		return nil, err
	}

	err = b.FoundAddr.AssertOverLap()
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Section struct {
	Name string
	Size uint64

	Offset uint64
	End    uint64

	Addr    uint64
	AddrEnd uint64

	OnlyInMemory bool
}

type File struct {
	Path      string
	Functions []*gore.Function
	Methods   []*gore.Method
}

func (f *File) GetSize() uint64 {
	var size uint64 = 0
	for _, fn := range f.Functions {
		size += fn.End - fn.Offset
	}
	return size
}
