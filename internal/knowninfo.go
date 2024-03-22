package internal

import (
	"debug/elf"
	"debug/gosym"
	"debug/macho"
	"debug/pe"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/goretk/gore"
	"log/slog"
	"os"
	"reflect"
	"unsafe"
)

type KnownInfo struct {
	Size       uint64
	BuildInfo  *gore.BuildInfo
	SectionMap *SectionMap
	Packages   *MainPackages
	KnownAddr  *KnownAddr

	Coverage AddrCoverage

	gore    *gore.GoFile
	wrapper wrapper.RawFileWrapper

	VersionFlag struct {
		Leq118 bool
		Meq120 bool
	}
}

func NewKnownInfo(file *gore.GoFile) *KnownInfo {
	// ensure we have the version
	k := &KnownInfo{
		Size:      utils.GetFileSize(file.GetFile()),
		BuildInfo: file.BuildInfo,

		gore:    file,
		wrapper: wrapper.NewWrapper(file.GetParsedFile()),
	}
	k.KnownAddr = NewKnownAddr(k)
	k.UpdateVersionFlag()

	return k
}

func (k *KnownInfo) LoadSectionMap() {
	slog.Info("Loading sections...")

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

	slog.Info("Loading sections done")

	k.SectionMap = sections

	return
}

func (k *KnownInfo) AnalyzeSymbol(file *gore.GoFile) error {
	slog.Info("Analyzing symbols...")
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

	k.KnownAddr.BuildSymbolCoverage()

	slog.Info("Analyzing symbols done")

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
	sym := &gosym.Sym{
		Name: s,
	}

	val := reflect.ValueOf(sym).Elem()
	ver := val.FieldByName("goVersion")

	set := func(i int) {
		reflect.NewAt(ver.Type(), unsafe.Pointer(ver.UnsafeAddr())).Elem().SetInt(int64(i))
	}

	if k.VersionFlag.Meq120 {
		set(5) // ver120
	} else if k.VersionFlag.Leq118 {
		set(4) // ver118
	}

	pn := sym.PackageName()

	return utils.UglyGuess(pn)
}

func (k *KnownInfo) GetPaddingSize() uint64 {
	var sectionSize uint64 = 0
	for _, section := range k.SectionMap.Sections {
		sectionSize += section.Size
	}
	return k.Size - sectionSize
}

func (k *KnownInfo) RequireModInfo() {
	if k.BuildInfo == nil {
		slog.Error("buildinfo is required for this operation")
		os.Exit(1)
	}
}

func (k *KnownInfo) CollectCoverage() {
	var load func(p *Package)
	load = func(p *Package) {
		// we always load leaf first
		for _, sp := range p.SubPackages {
			load(sp)
		}

		// then collect the coverage from the sub packages
		covers := make([]AddrCoverage, 0, len(p.SubPackages))
		for _, sp := range p.SubPackages {
			covers = append(covers, sp.coverage)
		}
		p.coverage = p.GetAddrSpace().GetCoverage(covers...)
	}

	// load coverage for all top packages
	for _, p := range k.Packages.topPkgs {
		load(p)
	}

	// load coverage for pclntab and symbol
	pclntabCov := k.KnownAddr.pclntab.GetCoverage()

	// merge all
	covs := make([]AddrCoverage, 0, len(k.Packages.topPkgs)+2)
	for _, p := range k.Packages.topPkgs {
		covs = append(covs, p.coverage)
	}
	covs = append(covs, pclntabCov, k.KnownAddr.symbolCoverage)
	k.Coverage = AddrSpace{}.GetCoverage(covs...)
}

func (k *KnownInfo) CalculateSectionSize() {
	for _, section := range k.SectionMap.Sections {
		size := uint64(0)
		for _, addr := range k.Coverage {
			if section.Addr <= addr.Addr && addr.Addr < section.Addr+section.Size {
				size += addr.Size
			}
		}
		section.KnownSize = size
	}
}

func (k *KnownInfo) CalculatePackageSize() {
	for _, p := range k.Packages.link {
		size := uint64(0)

		for _, addr := range p.coverage {
			size += addr.Size
		}
		for _, fn := range p.GetFunctions() {
			size += fn.Size
		}
		p.Size = size
	}
}
