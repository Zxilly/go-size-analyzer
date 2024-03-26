package internal

import (
	"debug/gosym"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/goretk/gore"
	"log/slog"
	"os"
	"reflect"
	"slices"
	"unsafe"
)

type KnownInfo struct {
	Size      uint64
	BuildInfo *gore.BuildInfo
	Sects     *SectionMap
	Deps      *Dependencies
	KnownAddr *entity.KnownAddr

	Coverage entity.AddrCoverage

	gore    *gore.GoFile
	wrapper wrapper.RawFileWrapper

	VersionFlag struct {
		Leq118 bool
		Meq120 bool
	}
}

func (k *KnownInfo) LoadSectionMap() {
	slog.Info("Loading sections...")

	sections := k.wrapper.LoadSections()

	slog.Info("Loading sections done")

	k.Sects = &SectionMap{
		Sections: sections,
	}

	return
}

func (k *KnownInfo) AnalyzeSymbol(file *gore.GoFile) error {
	slog.Info("Analyzing symbols...")

	err := k.wrapper.LoadSymbols(k.MarkSymbol)
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

func (k *KnownInfo) LoadPackages(file *gore.GoFile) error {
	slog.Info("Loading packages...")

	pkgs := NewDependencies(k)
	k.Deps = pkgs

	pclntab, err := file.PCLNTab()
	if err != nil {
		return err
	}

	self, err := file.GetPackages()
	if err != nil {
		return err
	}
	for _, p := range self {
		pkgs.Add(p, entity.PackageTypeMain, pclntab)
	}

	grStd, _ := file.GetSTDLib()
	for _, p := range grStd {
		pkgs.Add(p, entity.PackageTypeStd, pclntab)
	}

	grVendor, _ := file.GetVendors()
	for _, p := range grVendor {
		pkgs.Add(p, entity.PackageTypeVendor, pclntab)
	}

	grGenerated, _ := file.GetGeneratedPackages()
	for _, p := range grGenerated {
		pkgs.Add(p, entity.PackageTypeGenerated, pclntab)
	}

	grUnknown, _ := file.GetUnknown()
	for _, p := range grUnknown {
		pkgs.Add(p, entity.PackageTypeUnknown, pclntab)
	}

	k.RequireModInfo()
	modules := slices.Clone(k.BuildInfo.ModInfo.Deps)
	modules = append(modules, &k.BuildInfo.ModInfo.Main)
	pkgs.PushUpUnloadPacakge(modules)

	slog.Info("Loading packages done")

	return nil
}

func (k *KnownInfo) GetPaddingSize() uint64 {
	var sectionSize uint64 = 0
	for _, section := range k.Sects.Sections {
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
	// load coverage for pclntab and symbol
	pclntabCov := k.KnownAddr.Pclntab.ToCoverage()

	// merge all
	covs := make([]entity.AddrCoverage, 0, len(k.Deps.topPkgs)+2)
	for _, p := range k.Deps.topPkgs {
		covs = append(covs, p.GetCoverage())
	}
	covs = append(covs, pclntabCov, k.KnownAddr.SymbolCoverage)
	k.Coverage = entity.MergeCoverage(covs...)
}

func (k *KnownInfo) CalculateSectionSize() {
	for _, section := range k.Sects.Sections {
		size := uint64(0)
		for _, addr := range k.Coverage {
			// calculate the overlapped size
			start := max(section.Addr, addr.Addr)
			end := min(section.Addr+section.Size, addr.Addr+addr.Size)
			if start < end {
				size += end - start
			}
		}
		section.KnownSize = size
	}
}

func (k *KnownInfo) CalculatePackageSize() {
	for _, p := range k.Deps.link {
		size := uint64(0)

		for _, addr := range p.GetCoverage() {
			size += addr.Size
		}
		for _, fn := range p.GetFunctions(true) {
			size += fn.Size
		}
		p.Size = size
	}
}
