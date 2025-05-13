package entity

import (
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/ZxillyFork/gore"
	"github.com/ZxillyFork/gosym"

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
	PackageTypeCGO       PackageType = "cgo"
)

type Package struct {
	Name string      `json:"name"`
	Type PackageType `json:"type"`

	SubPackages PackageMap `json:"subPackages"`
	Files       []*File    `json:"files"`

	Size uint64 `json:"size"` // late filled

	// should not be used to calculate size,
	// since linker can create overlapped symbols.
	// relies on coverage.
	Symbols []*Symbol `json:"symbols"`

	ImportedBy []string `json:"importedBy,omitempty"`

	filesCache map[string]*File
	funcsCache map[string]*Function

	loaded bool // mean it comes from gore

	symbolAddrSpace AddrSpace
	coverageGetter  func() AddrCoverage
}

func NewPackage() *Package {
	p := &Package{
		SubPackages: make(map[string]*Package),
		Files:       make([]*File, 0),
		Symbols:     make([]*Symbol, 0),
		ImportedBy:  make([]string, 0),

		symbolAddrSpace: AddrSpace{},
		filesCache:      make(map[string]*File),
		funcsCache:      make(map[string]*Function),
	}
	p.coverageGetter = sync.OnceValue(p.buildPackageCoverage)
	return p
}

func NewPackageWithGorePackage(gp *gore.Package, name string, typ PackageType, pclntab *gosym.Table, getCodeSize func(function *gore.Function) uint64, isWasm bool) *Package {
	p := NewPackage()
	p.Name = utils.Deduplicate(name)
	p.Type = typ
	p.loaded = true

	getFunction := func(f *gore.Function) *Function {
		// fixme: pclntab size for wasm currently broken
		pclnSize := PclnSymbolSize{}
		if !isWasm {
			pclnSize = NewPclnSymbolSize(f.Func)
		}

		return &Function{
			Name:     utils.Deduplicate(f.Name),
			Addr:     f.Offset,
			CodeSize: getCodeSize(f),
			PclnSize: pclnSize,
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
	fileSeen := make(map[string]*File)

	for _, f := range p.Files {
		if old, ok := fileSeen[f.FilePath]; ok {
			funcSeen := make(map[string]*Function)
			for _, fn := range old.Functions {
				funcSeen[fn.Name] = fn
			}

			for _, fn := range f.Functions {
				if _, ok := funcSeen[fn.Name]; !ok {
					old.Functions = append(old.Functions, fn)
				}
			}
		} else {
			fileSeen[f.FilePath] = f
		}
	}

	p.Files = utils.Collect(maps.Values(fileSeen))
	p.filesCache = fileSeen

	p.funcsCache = make(map[string]*Function)
	for _, f := range p.Files {
		for _, fn := range f.Functions {
			p.funcsCache[fn.Name] = fn
		}
	}
}

func (p *Package) addFunction(path string, fn *Function) {
	file := p.getOrInitFile(path)
	file.Functions = append(file.Functions, fn)
}

func (p *Package) AddFuncIfNotExists(path string, fn *Function) bool {
	if _, ok := p.funcsCache[fn.Name]; !ok {
		p.addFunction(path, fn)
		p.funcsCache[fn.Name] = fn
		return true
	}
	return false
}

func (p *Package) getOrInitFile(s string) *File {
	if f, ok := p.filesCache[s]; ok {
		return f
	}

	f := &File{
		FilePath:  utils.Deduplicate(s),
		PkgName:   p.Name,
		Functions: make([]*Function, 0),
	}

	p.Files = append(p.Files, f)
	p.filesCache[f.FilePath] = f
	return f
}

// Merge p always hold an empty subpackage
func (p *Package) Merge(rp *Package) {
	if rp == nil {
		panic(errors.New("nil package"))
	}

	if rp.loaded && p.Name != rp.Name {
		panic(fmt.Errorf("package name not match %s %s", p.Name, rp.Name))
	}

	p.Files = append(p.Files, rp.Files...)
	// prevent duplicate files
	p.fileEnsureUnique()

	for k, v := range rp.SubPackages {
		p.SubPackages[k] = v
	}
}

func (p *Package) Functions(yield func(*Function) bool) {
	for _, f := range p.Files {
		for _, fn := range f.Functions {
			if !yield(fn) {
				return
			}
		}
	}
}

func (p *Package) GetDisasmAddrSpace() AddrSpace {
	spaces := make([]AddrSpace, 0)
	for f := range p.Functions {
		spaces = append(spaces, f.disasm)
	}
	return MergeAddrSpace(spaces...)
}

func (p *Package) GetFunctionSizeRecursive() uint64 {
	size := uint64(0)
	for f := range p.Functions {
		size += f.Size()
	}
	for _, sp := range p.SubPackages {
		size += sp.GetFunctionSizeRecursive()
	}
	return size
}

func (p *Package) GetPackageCoverage() AddrCoverage {
	return p.coverageGetter()
}

func (p *Package) buildPackageCoverage() AddrCoverage {
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

	return cov
}

func (p *Package) AssignPackageSize() {
	pkgSize := p.GetFunctionSizeRecursive()
	for _, cp := range p.GetPackageCoverage() {
		pkgSize += cp.Pos.Size
	}
	p.Size = pkgSize
}

func (p *Package) AddSymbol(symbol *Symbol, ap *Addr) {
	// first, load as coverage
	// no need to check the section type as it has been checked before
	p.symbolAddrSpace.Insert(ap)

	// then, add to the symbol list
	p.Symbols = append(p.Symbols, symbol)
}

func (p *Package) ClearCache() {
	p.filesCache = nil
	p.funcsCache = nil
}
