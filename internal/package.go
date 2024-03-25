package internal

import (
	"debug/gosym"
	"errors"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/goretk/gore"
	"log/slog"
	"runtime/debug"
	"slices"
	"strings"
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

// MainPackages a pres-udo package for the whole binary
type MainPackages struct {
	link PackageMap
	k    *KnownInfo

	topPkgs PackageMap
}

func NewMainPackages(k *KnownInfo) *MainPackages {
	return &MainPackages{
		topPkgs: make(PackageMap),
		link:    make(PackageMap),
		k:       k,
	}
}

func (m *MainPackages) GetPackage(name string) (*Package, bool) {
	p, ok := m.link[name]
	return p, ok
}

func (m *MainPackages) GetFunctions() []*Function {
	funcs := make([]*Function, 0)
	for _, p := range m.topPkgs {
		funcs = append(funcs, p.GetFunctions()...)
	}
	return funcs
}

func (m *MainPackages) MergeEmptyPacakge(modules []*debug.Module) {
	noTypPkgs := make([]*Package, 0)
	for _, module := range modules {
		parts := strings.Split(module.Path, "/")
		if len(parts) == 0 {
			continue
		}
		firstPart := parts[0]
		if _, ok := m.topPkgs[firstPart]; !ok {
			continue // can this happen?
		}
		p := m.topPkgs[firstPart]
		for _, part := range parts[1:] {
			if _, ok := p.SubPackages[part]; !ok {
				goto next
			}
			p = p.SubPackages[part]
		}
		p.pseudo = true
		p.Name = module.Path

		// after subpackages are loaded, need to determine a type for these
		noTypPkgs = append(noTypPkgs, p)
		// should also update link for this
		m.link[module.Path] = p

	next:
	}

	partMerge := func(part ...string) string {
		return strings.Join(part, "/")
	}

	var expand func(p *Package, part string) (shouldExpand bool, expanded PackageMap)
	expand = func(p *Package, part string) (bool, PackageMap) {
		newSubs := make(PackageMap)
		for subPart, subPackage := range p.SubPackages {
			shouldExpand, expanded := expand(subPackage, subPart)
			if !shouldExpand {
				newSubs[subPart] = subPackage
			} else {
				for ek, ev := range expanded {
					newSubs[partMerge(subPart, ek)] = ev
				}
			}
		}

		if p.loaded || p.pseudo {
			p.SubPackages = newSubs
			return false, nil
		} else {
			return true, newSubs
		}
	}

	newPackages := make(PackageMap)
	for part, p := range m.topPkgs {
		shouldExpand, expanded := expand(p, part)
		if shouldExpand {
			for k, v := range expanded {
				newPackages[partMerge(part, k)] = v
			}
		} else {
			newPackages[part] = p
		}
	}

	// We can load a type now
	for _, p := range noTypPkgs {
		if len(p.SubPackages) > 0 {
			typ, err := p.SubPackages.GetType()
			if err != nil {
				panic(fmt.Errorf("package %s has %s", p.Name, err))
			}
			if p.Type == "" {
				p.Type = typ
			} else if p.Type != typ {
				panic(fmt.Errorf("package %s has multiple type %s and %s", p.Name, p.Type, typ))
			}
		}
		if p.Type == "" {
			panic(fmt.Errorf("package %s has no type", p.Name))
		}
	}

	m.topPkgs = newPackages
}

func (m *MainPackages) Add(gp *gore.Package, typ PackageType, pclntab *gosym.Table) {
	name := gp.Name
	if typ == PackageTypeVendor {
		name = utils.UglyGuess(gp.Name)
	}

	parts := strings.Split(name, "/")

	if len(parts) == 0 {
		panic("empty package name " + gp.Name)
	}
	var container = m.topPkgs
	for i, p := range parts {
		if i == len(parts)-1 {
			break
		}

		if _, ok := container[p]; !ok {
			container[p] = NewPackage()
		}
		container = container[p].SubPackages
	}

	id := parts[len(parts)-1]

	p := NewPackageWithGorePackage(gp, name, typ, pclntab)

	// update addrs
	for _, f := range p.Functions {
		m.k.KnownAddr.InsertPclntab(f.Addr, f.Size, f, GoPclntabMeta{
			FuncName:    Deduplicate(f.Name),
			PackageName: Deduplicate(p.Name),
			Type:        Deduplicate(f.Type),
			Receiver:    Deduplicate(f.Receiver),
			Filepath:    Deduplicate(f.Filepath),
		})
	}

	p.Merge(container[id])

	container[id] = p
	// also update the link
	m.link[name] = p
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

	SubPackages PackageMap  `json:"subPackages"`
	Functions   []*Function `json:"functions,omitempty"`

	Size uint64 `json:"size"` // late filled

	coverage AddrCoverage

	loaded bool // mean it has the meaningful data
	pseudo bool // mean it's a pseudo package

	grPkg *gore.Package
}

func NewPackage() *Package {
	return &Package{
		SubPackages: make(map[string]*Package),
	}
}

func NewPackageWithGorePackage(gp *gore.Package, name string, typ PackageType, pclntab *gosym.Table) *Package {
	p := &Package{
		Name:        Deduplicate(name),
		Functions:   make([]*Function, 0, len(gp.Functions)+len(gp.Methods)),
		Type:        typ,
		loaded:      true,
		SubPackages: make(PackageMap),
		grPkg:       gp,
	}

	for _, f := range gp.Functions {
		src, _, _ := pclntab.PCToLine(f.Offset)
		p.Functions = append(p.Functions, &Function{
			Name:     Deduplicate(f.Name),
			Addr:     f.Offset,
			Size:     f.End - f.Offset,
			Type:     FuncTypeFunction,
			Receiver: Deduplicate(""),
			Filepath: Deduplicate(src),
			Disasm:   AddrSpace{},
			Pkg:      p,
		})
	}
	for _, mf := range gp.Methods {
		src, _, _ := pclntab.PCToLine(mf.Offset)
		p.Functions = append(p.Functions, &Function{
			Name:     Deduplicate(mf.Name),
			Addr:     mf.Offset,
			Size:     mf.End - mf.Offset,
			Type:     FuncTypeMethod,
			Receiver: Deduplicate(mf.Receiver),
			Filepath: Deduplicate(src),
			Disasm:   AddrSpace{},
			Pkg:      p,
		})
	}

	return p
}

// Merge p always hold an empty subpackage
func (p *Package) Merge(rp *Package) {
	if rp == nil {
		return
	}

	if (rp.loaded || rp.pseudo) && p.Name != rp.Name {
		panic(fmt.Errorf("package name not match %s %s", p.Name, rp.Name))
	}

	for _, f := range rp.Functions {
		p.Functions = append(p.Functions, f)
	}
	for k, v := range rp.SubPackages {
		p.SubPackages[k] = v
	}

}

func (p *Package) GetFunctions() []*Function {
	funcs := make([]*Function, 0, len(p.Functions))
	for _, f := range p.Functions {
		funcs = append(funcs, f)
	}
	for _, sp := range p.SubPackages {
		funcs = append(funcs, sp.GetFunctions()...)
	}
	return funcs
}

func (p *Package) GetAddrSpace() AddrSpace {
	ret := AddrSpace{}
	for _, f := range p.Functions {
		ret.Merge(f.Disasm)
	}
	return ret
}

func (k *KnownInfo) LoadPackages(file *gore.GoFile) error {
	slog.Info("Loading packages...")

	pkgs := NewMainPackages(k)
	k.Packages = pkgs

	pclntab, err := file.PCLNTab()
	if err != nil {
		return err
	}

	self, err := file.GetPackages()
	if err != nil {
		return err
	}
	for _, p := range self {
		pkgs.Add(p, PackageTypeMain, pclntab)
	}

	grStd, _ := file.GetSTDLib()
	for _, p := range grStd {
		pkgs.Add(p, PackageTypeStd, pclntab)
	}

	grVendor, _ := file.GetVendors()
	for _, p := range grVendor {
		pkgs.Add(p, PackageTypeVendor, pclntab)
	}

	grGenerated, _ := file.GetGeneratedPackages()
	for _, p := range grGenerated {
		pkgs.Add(p, PackageTypeGenerated, pclntab)
	}

	grUnknown, _ := file.GetUnknown()
	for _, p := range grUnknown {
		pkgs.Add(p, PackageTypeUnknown, pclntab)
	}

	k.RequireModInfo()
	modules := slices.Clone(k.BuildInfo.ModInfo.Deps)
	modules = append(modules, &k.BuildInfo.ModInfo.Main)
	pkgs.MergeEmptyPacakge(modules)

	slog.Info("Loading packages done")

	return nil
}
