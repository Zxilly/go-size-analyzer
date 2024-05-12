package entity

import (
	"fmt"
	"runtime/debug"

	"github.com/ZxillyFork/gore"
	"github.com/ZxillyFork/gosym"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

type PackageMap map[string]*Package

type PackageType = string

const (
	PackageTypeMain      PackageType = "main"
	PackageTypeStd       PackageType = "std"
	PackageTypeVendor    PackageType = "vendor"
	PackageTypeGenerated PackageType = "generated"
	PackageTypeUnknown   PackageType = "unknown"
)

type Package struct {
	Name string      `json:"name"`
	Type PackageType `json:"type"`

	SubPackages PackageMap `json:"subPackages"`
	Files       []*File    `json:"files"`

	Size uint64 `json:"size"` // late filled

	loaded bool // mean it comes from gore

	// should not be used to calculate size,
	// since linker can create overlapped symbols.
	// relies on coverage.
	// currently only data symbol
	Symbols []*Symbol `json:"symbols"`

	symbolAddrSpace AddrSpace
	coverage        *utils.ValueOnce[AddrCoverage]

	// should have at least one of them
	GorePkg  *gore.Package `json:"-"`
	DebugMod *debug.Module `json:"-"`
}

func NewPackage() *Package {
	return &Package{
		SubPackages:     make(map[string]*Package),
		Files:           make([]*File, 0),
		Symbols:         make([]*Symbol, 0),
		coverage:        utils.NewOnce[AddrCoverage](),
		symbolAddrSpace: AddrSpace{},
	}
}

func NewPackageWithGorePackage(gp *gore.Package, name string, typ PackageType, pclntab *gosym.Table) *Package {
	p := NewPackage()
	p.Name = utils.Deduplicate(name)
	p.Type = typ
	p.loaded = true
	p.GorePkg = gp

	getFunction := func(f *gore.Function) *Function {
		return &Function{
			Name:     utils.Deduplicate(f.Name),
			Addr:     f.Offset,
			CodeSize: f.End - f.Offset,
			PclnSize: NewPclnSymbolSize(f.Func),
			Type:     FuncTypeFunction,
			disasm:   AddrSpace{},
			pkg:      p,
		}
	}

	for _, f := range gp.Functions {
		src, _, _ := pclntab.PCToLine(f.Offset)
		sf := getFunction(f)
		sf.Type = FuncTypeFunction
		p.addFunction(src, sf)
	}
	for _, mf := range gp.Methods {
		src, _, _ := pclntab.PCToLine(mf.Offset)
		sf := getFunction(mf.Function)
		sf.Type = FuncTypeMethod
		sf.Receiver = utils.Deduplicate(mf.Receiver)
		p.addFunction(src, sf)
	}

	return p
}

func (p *Package) fileEnsureUnique() {
	seen := make(map[string]*File)
	for _, f := range p.Files {
		if old, ok := seen[f.FilePath]; ok {
			old.Functions = append(old.Functions, f.Functions...)
		} else {
			seen[f.FilePath] = f
		}
	}
	p.Files = maps.Values(seen)
}

func (p *Package) addFunction(path string, fn *Function) {
	file := p.getOrInitFile(path)

	fn.File = file

	file.Functions = append(file.Functions, fn)
}

func (p *Package) getOrInitFile(s string) *File {
	for _, f := range p.Files {
		if f.FilePath == s {
			return f
		}
	}

	f := &File{
		FilePath:  utils.Deduplicate(s),
		Pkg:       p,
		Functions: make([]*Function, 0),
	}

	p.Files = append(p.Files, f)
	return f
}

// Merge p always hold an empty subpackage
func (p *Package) Merge(rp *Package) {
	if rp == nil {
		panic(fmt.Errorf("nil package"))
	}

	if (rp.loaded) && p.Name != rp.Name {
		panic(fmt.Errorf("package name not match %s %s", p.Name, rp.Name))
	}

	p.Files = append(p.Files, rp.Files...)
	// prevent duplicate files
	p.fileEnsureUnique()

	for k, v := range rp.SubPackages {
		p.SubPackages[k] = v
	}
}

func (p *Package) GetFunctions() []*Function {
	funcs := lo.Reduce(p.Files,
		func(agg []*Function, item *File, _ int) []*Function {
			return append(agg, item.Functions...)
		}, make([]*Function, 0))

	return funcs
}

func (p *Package) GetDisasmAddrSpace() AddrSpace {
	spaces := make([]AddrSpace, 0)
	for _, f := range p.GetFunctions() {
		spaces = append(spaces, f.disasm)
	}
	return MergeAddrSpace(spaces...)
}

func (p *Package) GetFunctionSizeRecursive() uint64 {
	size := uint64(0)
	for _, f := range p.GetFunctions() {
		size += f.Size()
	}
	for _, sp := range p.SubPackages {
		size += sp.GetFunctionSizeRecursive()
	}
	return size
}

func (p *Package) GetPackageCoverage() AddrCoverage {
	p.coverage.Do(func() {
		disasmcov := p.GetDisasmAddrSpace().ToDirtyCoverage()
		symbolcov := p.symbolAddrSpace.ToDirtyCoverage()

		covs := []AddrCoverage{disasmcov, symbolcov}

		for _, sp := range p.SubPackages {
			covs = append(covs, sp.GetPackageCoverage())
		}

		cov, err := MergeAndCleanCoverage(covs)
		if err != nil {
			panic(err)
		}

		p.coverage.Set(cov)
	})
	return p.coverage.Get()
}

func (p *Package) AssignPackageSize() {
	pkgSize := p.GetFunctionSizeRecursive()
	for _, cp := range p.GetPackageCoverage() {
		pkgSize += cp.Pos.Size
	}
	p.Size = pkgSize
}

func (p *Package) AddSymbol(addr uint64, size uint64, typ AddrType, name string, ap *Addr) {
	// first, load as coverage
	p.symbolAddrSpace.Insert(ap)

	// then, add to the symbol list
	p.Symbols = append(p.Symbols, NewSymbol(name, addr, size, typ))
}
