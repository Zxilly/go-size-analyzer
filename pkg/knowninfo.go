package pkg

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
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
	KnownAddr  *KnownAddr

	gore *gore.GoFile

	VersionFlag struct {
		Leq118 bool
		Meq120 bool
	}
}

func NewKnownInfo(file *gore.GoFile) *KnownInfo {
	// ensure we have the version
	k := &KnownInfo{
		KnownAddr: NewFoundAddr(),
		Size:      tool.GetFileSize(file.GetFile()),
		BuildInfo: file.BuildInfo,

		gore: file,
	}
	k.UpdateVersionFlag()
	return k
}

func (k *KnownInfo) LoadSectionMap() {
	log.Println("Loading sections...")

	sections := &SectionMap{Sections: make(map[string]*Section)}

	switch f := k.gore.GetParsedFile().(type) {
	case *pe.File:
		sections.loadFromPe(f)
	case *elf.File:
		sections.loadFromElf(f)
	case *macho.File:
		sections.loadFromMacho(f)
	default:
		panic("unreachable")
	}

	log.Println("Loading sections done")

	k.SectionMap = sections

	return
}

func (k *KnownInfo) AnalyzeSymbol(file *gore.GoFile) error {
	log.Println("Analyzing symbols...")
	var err error

	switch f := file.GetParsedFile().(type) {
	case *pe.File:
		err = analyzePeSymbol(f, k)
	case *elf.File:
		err = analyzeElfSymbol(f, k)
	case *macho.File:
		err = analyzeMachoSymbol(f, k)
	default:
		panic("unreachable")
	}

	if err != nil {
		return err
	}

	log.Println("Analyzing symbols done")

	return nil
}

func (k *KnownInfo) Validate() error {
	return k.KnownAddr.Validate()
}

func (k *KnownInfo) MarkSymbol(name string, addr, size uint64, typ AddrType) error {
	pkgName := k.ExtractPackageFromSymbol(name)
	if pkgName == "" {
		return nil // no package or compiler-generated symbol, skip
	}

	pkg, ok := k.Packages.NameToPkg[pkgName]
	if !ok {
		return nil // no package found, skip
	}

	k.KnownAddr.Insert(addr, size, pkg, AddrSourceSymbol, typ, SymbolMeta{
		SymbolName:  Deduplicate(name),
		PackageName: Deduplicate(pkgName),
	})

	return nil
}

func (k *KnownInfo) UpdateVersionFlag() {
	ver, err := k.gore.GetCompilerVersion()
	if err != nil {
		// if we can't get build info, we assume it's go1.20 plus
		k.VersionFlag.Meq120 = true
	} else {
		k.VersionFlag.Leq118 = gore.GoVersionCompare(ver.Name, "go1.18.10") <= 0
		k.VersionFlag.Meq120 = gore.GoVersionCompare(ver.Name, "go1.20rc1") >= 0
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
