package entity

import (
	"errors"
	"fmt"
	"maps"
	"strings"
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
	funcsCache map[uint64]*Function

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
		funcsCache:      make(map[uint64]*Function),
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
		if strings.HasPrefix(f.Name, "go:") {
			continue
		}
		src, _, _ := pclntab.PCToLine(f.Offset)
		sf := getFunction(f)
		sf.Type = FuncTypeFunction
		p.AddFuncIfNotExists(src, sf)
	}
	for _, mf := range gp.Methods {
		src, _, _ := pclntab.PCToLine(mf.Offset)
		sf := getFunction(mf.Function)
		sf.Type = FuncTypeMethod
		sf.Receiver = utils.Deduplicate(mf.Receiver)
		p.AddFuncIfNotExists(src, sf)
	}

	return p
}

func (p *Package) fileEnsureUnique() {
	files := p.Files
	p.Files = nil
	p.filesCache = make(map[string]*File)
	p.funcsCache = make(map[uint64]*Function)
	for _, file := range files {
		for _, fn := range file.Functions {
			p.AddFuncIfNotExists(file.FilePath, fn)
		}
	}
}
func (p *Package) addFunction(path string, fn *Function) {
	fn.pkg = p
	file := p.getOrInitFile(path)
	file.Functions = append(file.Functions, fn)
	p.funcsCache[fn.Addr] = fn
}

func (p *Package) AddFuncIfNotExists(path string, fn *Function) bool {
	if _, ok := p.funcsCache[fn.Addr]; !ok {
		p.addFunction(path, fn)
		p.funcsCache[fn.Addr] = fn
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

	maps.Copy(p.SubPackages, rp.SubPackages)
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

// AddSymbolCoverage registers the address range for package-size accounting
// without appending to the visible Symbols list. Use when many tiny ranges
// should share a single aggregated display symbol that the caller emits
// separately via AddSymbol.
func (p *Package) AddSymbolCoverage(ap *Addr) {
	p.symbolAddrSpace.Insert(ap)
}

func (p *Package) ClearCache() {
	p.filesCache = nil
	p.funcsCache = nil
}

// FuncCount returns the number of functions in this package
func (p *Package) FuncCount() int {
	return len(p.funcsCache)
}
