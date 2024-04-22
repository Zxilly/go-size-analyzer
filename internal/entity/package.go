package entity

import (
	"debug/gosym"
	"errors"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/goretk/gore"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

type PackageMap map[string]*Package

func (m PackageMap) GetType() (typ PackageType, err error) {
	load := func(packageType PackageType) error {
		if packageType == "" {
			return nil
		}

		if typ == "" {
			typ = packageType
			return nil
		} else {
			if typ != packageType {
				return errors.New("multiple package type found")
			}
		}
		return nil
	}

	for _, p := range m {
		err = load(p.Type)
		if err != nil {
			return
		}
		if len(p.SubPackages) > 0 {
			var st PackageType
			st, err = p.SubPackages.GetType()
			if err != nil {
				return
			}
			err = load(st)
			if err != nil {
				return
			}
		}
	}

	if typ == "" {
		return "", errors.New("no package type found")
	}

	return
}

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

	Loaded bool `json:"-"` // mean it has the meaningful data
	Pseudo bool `json:"-"` // mean it's a Pseudo package

	disAsmCoverage *utils.ValueOnce[AddrCoverage]

	grPkg *gore.Package
}

func NewPackage() *Package {
	return &Package{
		SubPackages:    make(map[string]*Package),
		Files:          make([]*File, 0),
		disAsmCoverage: utils.NewOnce[AddrCoverage](),
	}
}

func NewPackageWithGorePackage(gp *gore.Package, name string, typ PackageType, pclntab *gosym.Table) *Package {
	p := NewPackage()
	p.Name = utils.Deduplicate(name)
	p.Type = typ
	p.Loaded = true
	p.grPkg = gp

	for _, f := range gp.Functions {
		src, _, _ := pclntab.PCToLine(f.Offset)
		p.addFunction(src, &Function{
			Name:     utils.Deduplicate(f.Name),
			Addr:     f.Offset,
			Size:     f.End - f.Offset,
			Type:     FuncTypeFunction,
			Receiver: utils.Deduplicate(""),
			disasm:   AddrSpace{},
			pkg:      p,
		})
	}
	for _, mf := range gp.Methods {
		src, _, _ := pclntab.PCToLine(mf.Offset)
		p.addFunction(src, &Function{
			Name:     utils.Deduplicate(mf.Name),
			Addr:     mf.Offset,
			Size:     mf.End - mf.Offset,
			Type:     FuncTypeMethod,
			Receiver: utils.Deduplicate(mf.Receiver),
			disasm:   AddrSpace{},
			pkg:      p,
		})
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
		pkg:       p,
		Functions: make([]*Function, 0),
	}

	p.Files = append(p.Files, f)
	return f
}

// Merge p always hold an empty subpackage
func (p *Package) Merge(rp *Package) {
	if rp == nil {
		return
	}

	if (rp.Loaded || rp.Pseudo) && p.Name != rp.Name {
		panic(fmt.Errorf("package name not match %s %s", p.Name, rp.Name))
	}

	for _, f := range rp.Files {
		p.Files = append(p.Files, f)
	}
	// prevent duplicate files
	p.fileEnsureUnique()

	for k, v := range rp.SubPackages {
		p.SubPackages[k] = v
	}

}

func (p *Package) GetFunctions(recursive bool) []*Function {
	funcs := make([]*Function, 0)
	for _, file := range p.Files {
		for _, fn := range file.Functions {
			funcs = append(funcs, fn)
		}
	}
	if recursive {
		for _, sp := range p.SubPackages {
			funcs = append(funcs, sp.GetFunctions(true)...)
		}
	}
	return funcs
}

func (p *Package) GetAddrSpace() AddrSpace {
	spaces := make([]AddrSpace, 0)
	for _, f := range p.GetFunctions(false) {
		spaces = append(spaces, f.disasm)
	}
	return MergeAddrSpace(spaces...)
}

func (p *Package) GetCoverage() AddrCoverage {
	p.disAsmCoverage.Do(func() {
		this := p.GetAddrSpace().ToDirtyCoverage()
		subs := lo.Map(maps.Values(p.SubPackages), func(p *Package, _ int) AddrCoverage {
			return p.GetCoverage()
		})
		merged, _ := MergeCoverage(append(subs, this))
		p.disAsmCoverage.Set(merged)
	})
	return p.disAsmCoverage.Get()
}
