package internal

import (
	"debug/gosym"
	"errors"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/goretk/gore"
	"log"
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

	Packages PackageMap
}

func NewMainPackages(k *KnownInfo) *MainPackages {
	return &MainPackages{
		Packages: make(PackageMap),
		link:     make(PackageMap),
		k:        k,
	}
}

func (m *MainPackages) GetPackage(name string) (*Package, bool) {
	p, ok := m.link[name]
	return p, ok
}

func (m *MainPackages) GetFunctions() []*Function {
	funcs := make([]*Function, 0)
	for _, p := range m.Packages {
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
		if _, ok := m.Packages[firstPart]; !ok {
			continue // can this happen?
		}
		p := m.Packages[firstPart]
		for _, part := range parts[1:] {
			if _, ok := p.SubPackages[part]; !ok {
				goto next
			}
			p = p.SubPackages[part]
		}
		p.pseudo = true
		p.Name = module.Path

		// after subpackages are load, need to determine type for these
		noTypPkgs = append(noTypPkgs, p)
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
	for part, p := range m.Packages {
		shouldExpand, expanded := expand(p, part)
		if shouldExpand {
			for k, v := range expanded {
				newPackages[partMerge(part, k)] = v
			}
		} else {
			newPackages[part] = p
		}
	}

	// We can load type now
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

	m.Packages = newPackages
}

func (m *MainPackages) Add(gp *gore.Package, typ PackageType, pclntab *gosym.Table) {
	name, err := utils.PrefixToPath(gp.Name)
	if err != nil {
		panic(err)
	}

	parts := strings.Split(name, "/")

	// an ugly hack for a known issue about golang compiler
	// sees https://github.com/golang/go/issues/66313
	if strings.Count(name, ".") >= 3 {
		// we met something like
		// github.com/ZNotify/server/app/api/common.(*Context).github.com/gin-gonic/gin
		// no way to process this kind of package as for now
		return
	}

	if len(parts) == 0 {
		panic("empty package name " + gp.Name)
	}
	var container = m.Packages
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
	subs := make(PackageMap)
	if c, ok := container[id]; ok {
		if c.loaded {
			panic("duplicate package " + name)
		}
		subs = c.SubPackages
	}
	p := &Package{
		Name:        Deduplicate(name),
		Functions:   make([]*Function, 0, len(gp.Functions)+len(gp.Methods)),
		Type:        typ,
		SubPackages: subs,
		loaded:      true,
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

	container[id] = p

	// also update the link
	m.link[gp.Name] = p

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
}

type PackageType = string

const (
	PackageTypeSelf      PackageType = "self"
	PackageTypeStd       PackageType = "std"
	PackageTypeVendor    PackageType = "vendor"
	PackageTypeGenerated PackageType = "generated"
	PackageTypeUnknown   PackageType = "unknown"
)

type Package struct {
	Name        string      `json:"name"`
	Functions   []*Function `json:"functions"`
	Type        PackageType `json:"type"`
	SubPackages PackageMap  `json:"subPackages"`

	loaded bool // mean it has the meaningful data
	pseudo bool // mean it's a pseudo package

	grPkg *gore.Package
}

func NewPackage() *Package {
	return &Package{
		SubPackages: make(map[string]*Package),
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

func (k *KnownInfo) LoadPackages(file *gore.GoFile) error {
	log.Println("Loading packages...")

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
		pkgs.Add(p, PackageTypeSelf, pclntab)
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

	log.Println("Loading packages done")

	return nil
}
