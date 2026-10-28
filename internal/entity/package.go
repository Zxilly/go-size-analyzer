package entity

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
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

	Size         uint64 `json:"size"`                   // late filled
	MetadataSize uint64 `json:"metadata_size,omitzero"` // file metadata owned by this package, excluding children

	// should not be used to calculate size,
	// since linker can create overlapped symbols.
	// relies on coverage.
	Symbols []*Symbol `json:"symbols"`

	ImportedBy []string `json:"importedBy,omitempty"`

	filesCache map[string]*File
	funcsCache map[uint64]*Function

	loaded bool // mean it comes from gore

	symbolAddrSpace AddrSpace
	coverageGetter  func() (AddrCoverage, error)
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
	p.coverageGetter = sync.OnceValues(p.buildPackageCoverage)
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
			Wasm:     isWasm,
			Source:   AddrSourceGoPclntab,
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

// GetFunction finds a function registered during analysis without replacing
// its display name, source file, or original code size.
func (p *Package) GetFunction(addr uint64) (*Function, bool) {
	fn, ok := p.funcsCache[addr]
	return fn, ok
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

func (p *Package) GetPackageCoverage() (AddrCoverage, error) {
	return p.coverageGetter()
}

func (p *Package) buildPackageCoverage() (AddrCoverage, error) {
	var own AddrCoverage
	for addr := range p.OwnAddresses {
		own = append(own, &CoveragePart{Pos: addr.AddrPos, Addrs: []*Addr{addr}})
	}
	covs := []AddrCoverage{own}

	for _, sp := range p.SubPackages {
		cov, err := sp.GetPackageCoverage()
		if err != nil {
			return nil, err
		}
		covs = append(covs, cov)
	}

	cov, err := MergeAndCleanCoverage(covs)
	if err != nil {
		return nil, fmt.Errorf("package %q coverage: %w", p.Name, err)
	}

	return cov, nil
}

func (p *Package) AssignPackageSize() error {
	cov, err := p.GetPackageCoverage()
	if err != nil {
		return err
	}
	pkgSize := uint64(0)
	var wasmSize func(*Package)
	wasmSize = func(pkg *Package) {
		for fn := range pkg.Functions {
			if fn.Wasm {
				pkgSize += fn.CodeSize
			}
		}
		for _, sub := range pkg.SubPackages {
			wasmSize(sub)
		}
	}
	wasmSize(p)
	for _, cp := range cov {
		pkgSize += cp.Pos.Size
	}
	p.Size = pkgSize
	return nil
}

// OwnAddresses preserves unmerged evidence for global file accounting.
func (p *Package) OwnAddresses(yield func(*Addr) bool) {
	var snapshots []addrSnapshot
	p.visitOwnAddresses(yield, func(pos AddrPos, fn *Function, source AddrSourceType) bool {
		if snapshots == nil {
			// A retained yield pointer must keep its own slot. Reserve the usual
			// total once, while append still permits callbacks to add more ranges.
			var count int
			for f := range p.Functions {
				count += len(f.PclnRanges)
				if !f.Wasm && len(f.codeEvidence) == 0 {
					count += max(1, len(f.Ranges))
				}
			}
			snapshots = make([]addrSnapshot, 0, count)
		}
		snapshots = append(snapshots, addrSnapshot{})
		snapshot := &snapshots[len(snapshots)-1]
		snapshot.pos = pos
		snapshot.addr = Addr{AddrPos: &snapshot.pos, Pkg: p, Function: fn, SourceType: source}
		snapshot.addr.setSourceEvidence()
		return yield(&snapshot.addr)
	})
}

// OwnAddressValues visits the same unmerged evidence as OwnAddresses without
// allocating pointer snapshots for synthetic code and pclntab ranges. Stored
// records are copied by value, and stopping the callback stops the traversal.
func (p *Package) OwnAddressValues(yield func(AddrValue) bool) {
	p.visitOwnAddresses(func(addr *Addr) bool {
		return yield(AddrValue{AddrPos: *addr.AddrPos, Pkg: addr.Pkg, Function: addr.Function, Symbol: addr.Symbol, SourceType: addr.SourceType, Role: addr.Role, Boundary: addr.Boundary})
	}, func(pos AddrPos, fn *Function, source AddrSourceType) bool {
		role, boundary := sourceEvidence(source, "")
		return yield(AddrValue{AddrPos: pos, Pkg: p, Function: fn, SourceType: source, Role: role, Boundary: boundary})
	})
}

// Stored evidence already has stable addresses. Synthetic ranges are passed by
// value so each public iterator can choose the representation its caller needs.
func (p *Package) visitOwnAddresses(stored func(*Addr) bool, synthetic func(AddrPos, *Function, AddrSourceType) bool) {
	for addr := range p.symbolAddrSpace.All {
		if !stored(addr) {
			return
		}
	}
	var functionCount int
	for _, f := range p.Files {
		functionCount += len(f.Functions)
	}
	functions := make([]*Function, 0, functionCount)
	for _, f := range p.Files {
		functions = append(functions, f.Functions...)
	}
	slices.SortFunc(functions, func(a, b *Function) int {
		if n := cmp.Compare(a.Addr, b.Addr); n != 0 {
			return n
		}
		return cmp.Compare(a.Name, b.Name)
	})
	for _, fn := range functions {
		for addr := range fn.disasm.All {
			if !stored(addr) {
				return
			}
		}
		if !fn.Wasm {
			if len(fn.codeEvidence) > 0 {
				for addr := range fn.codeEvidence.All {
					if !stored(addr) {
						return
					}
				}
			} else {
				for r := range fn.CodeRegions {
					r.Type = AddrTypeText
					if !synthetic(r, fn, fn.Source) {
						return
					}
				}
			}
		}
		for _, r := range fn.PclnRanges {
			r.Type = AddrTypeData
			if !synthetic(r, fn, AddrSourceGoPclntab) {
				return
			}
		}
	}
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

func (p *Package) SymbolAddresses(yield func(*Addr) bool) {
	for addr := range p.symbolAddrSpace.All {
		if !yield(addr) {
			return
		}
	}
}

func (p *Package) ClearCache() {
	p.filesCache = nil
	p.funcsCache = nil
}

// FuncCount returns the number of functions in this package
func (p *Package) FuncCount() int {
	return len(p.funcsCache)
}
