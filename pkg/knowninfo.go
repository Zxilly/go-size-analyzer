package pkg

import (
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

	VersionFlag struct {
		Leq118 bool
		Meq120 bool
	}
}

func (k *KnownInfo) updateVersionFlag() {
	if k.BuildInfo != nil && k.BuildInfo.Compiler != nil {
		k.VersionFlag.Leq118 = gore.GoVersionCompare(k.BuildInfo.Compiler.Name, "go1.18") <= 0
		k.VersionFlag.Meq120 = gore.GoVersionCompare(k.BuildInfo.Compiler.Name, "go1.20") >= 0
	} else {
		// if we can't get build info, we assume it's go1.20 plus
		k.VersionFlag.Meq120 = true
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
