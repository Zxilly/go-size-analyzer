package internal

import (
	"debug/gosym"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/goretk/gore"
	"log/slog"
	"math"
	"reflect"
	"runtime/debug"
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
	err := k.Sects.AssertSize(k.Size)
	if err != nil {
		utils.FatalError(err)
	}

	return
}

func (k *KnownInfo) AnalyzeSymbol() error {
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

func (k *KnownInfo) LoadPackages() error {
	slog.Info("Loading packages...")

	pkgs := NewDependencies(k)
	k.Deps = pkgs

	pclntab, err := k.gore.PCLNTab()
	if err != nil {
		return err
	}

	self, err := k.gore.GetPackages()
	if err != nil {
		return err
	}
	for _, p := range self {
		pkgs.Add(p, entity.PackageTypeMain, pclntab)
	}

	grStd, _ := k.gore.GetSTDLib()
	for _, p := range grStd {
		pkgs.Add(p, entity.PackageTypeStd, pclntab)
	}

	grVendor, _ := k.gore.GetVendors()
	for _, p := range grVendor {
		pkgs.Add(p, entity.PackageTypeVendor, pclntab)
	}

	grGenerated, _ := k.gore.GetGeneratedPackages()
	for _, p := range grGenerated {
		pkgs.Add(p, entity.PackageTypeGenerated, pclntab)
	}

	grUnknown, _ := k.gore.GetUnknown()
	for _, p := range grUnknown {
		pkgs.Add(p, entity.PackageTypeUnknown, pclntab)
	}

	k.RequireModInfo()

	pkgs.AddModules(k.BuildInfo.ModInfo.Deps, entity.PackageTypeVendor)
	pkgs.AddModules([]*debug.Module{&k.BuildInfo.ModInfo.Main}, entity.PackageTypeVendor)

	pkgs.FinishLoad()

	slog.Info("Loading packages done")

	return nil
}

func (k *KnownInfo) RequireModInfo() {
	if k.BuildInfo == nil {
		utils.FatalError(fmt.Errorf("no build info"))
	}
}

func (k *KnownInfo) CollectCoverage() {
	// load coverage for pclntab and symbol
	pclntabCov := k.KnownAddr.Pclntab.ToDirtyCoverage()

	// merge all
	covs := make([]entity.AddrCoverage, 0)

	// collect packages coverage
	_ = k.Deps.trie.Walk(func(_ string, value interface{}) error {
		p := value.(*entity.Package)
		covs = append(covs, p.GetPackageCoverage())
		return nil
	})

	covs = append(covs, pclntabCov, k.KnownAddr.SymbolCoverage)

	var err error
	k.Coverage, err = entity.MergeAndCleanCoverage(covs)
	if err != nil {
		utils.FatalError(err)
	}
}

func (k *KnownInfo) CalculateSectionSize() {
	t := make(map[*entity.Section]uint64)
	for _, cp := range k.Coverage {
		section := k.Sects.FindSection(cp.Pos.Addr, cp.Pos.Size)
		if section == nil {
			slog.Debug(fmt.Sprintf("section not found for coverage part %s", cp))
			continue
		}
		t[section] += cp.Pos.Size
	}

	for section, size := range t {
		mapper := 1.0
		if section.Size != section.FileSize {
			// need to map to file size
			mapper = float64(section.FileSize) / float64(section.Size)
		}
		section.KnownSize = uint64(math.Floor(float64(size) * mapper))
	}
}

// CalculatePackageSize calculate the size of each package
// Happens after disassembly
func (k *KnownInfo) CalculatePackageSize() {
	var dive func(p *entity.Package)
	dive = func(p *entity.Package) {
		if len(p.SubPackages) > 0 {
			for _, sp := range p.SubPackages {
				dive(sp)
			}
		}
		p.AssignPackageSize()
	}
	for _, p := range k.Deps.TopPkgs {
		dive(p)
	}
}
