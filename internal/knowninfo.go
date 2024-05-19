package internal

import (
	"fmt"
	"log/slog"
	"math"
	"runtime/debug"

	"github.com/ZxillyFork/gore"
	"github.com/ZxillyFork/gosym"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
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
	var ver gosym.Version
	if k.VersionFlag.Meq120 {
		ver = gosym.Ver120 // ver120
	} else if k.VersionFlag.Leq118 {
		ver = gosym.Ver118 // ver118
	}

	sym := &gosym.Sym{
		Name:      s,
		GoVersion: ver,
	}

	packageName := sym.PackageName()

	return utils.UglyGuess(packageName)
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
	_ = k.Deps.trie.Walk(func(_ string, p *entity.Package) error {
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
	// minus coverage part
	for _, cp := range k.Coverage {
		section := k.Sects.FindSection(cp.Pos.Addr, cp.Pos.Size)
		if section == nil {
			slog.Debug(fmt.Sprintf("section not found for coverage part %s", cp))
			continue
		}
		t[section] += cp.Pos.Size
	}

	pclntabSize := uint64(0)
	_ = k.Deps.trie.Walk(func(_ string, p *entity.Package) error {
		for _, fn := range p.GetFunctions() {
			pclntabSize += fn.PclnSize.Size()
		}
		return nil
	})

	// minus pclntab size
	possibleNames := k.wrapper.PclntabSections()
	for name, section := range k.Sects.Sections {
		for _, possibleName := range possibleNames {
			if possibleName == name {
				t[section] += pclntabSize
				goto foundPclntab
			}
		}
	}
	utils.FatalError(fmt.Errorf("pclntab section not found when calculate known size"))
foundPclntab:

	// linear map virtual size to file size
	for section, size := range t {
		mapper := 1.0
		if section.Size != section.FileSize {
			// need to map to file size
			mapper = float64(section.FileSize) / float64(section.Size)
		}
		section.KnownSize = uint64(math.Floor(float64(size) * mapper))

		if section.KnownSize > section.FileSize {
			// fixme: pclntab size calculation is not accurate
			slog.Warn(fmt.Sprintf("section %s known size %d > file size %d, this is a known issue", section.Name, section.KnownSize, section.FileSize))
			section.KnownSize = section.FileSize
		}
	}
}

// CalculatePackageSize calculate the size of each package
// Happens after disassembly
func (k *KnownInfo) CalculatePackageSize() {
	_ = k.Deps.trie.Walk(func(_ string, p *entity.Package) error {
		p.AssignPackageSize()
		return nil
	})
}
