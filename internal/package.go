package internal

import (
	"debug/gosym"
	"github.com/goretk/gore"
	"log"
	"runtime/debug"
	"slices"
	"strings"
)

type PackageMap map[string]*Package

// MainPackages a pres-udo package for the whole binary
type MainPackages struct {
	link     PackageMap
	Packages PackageMap
}

func NewMainPackages() *MainPackages {
	return &MainPackages{
		Packages: make(PackageMap),
		link:     make(PackageMap),
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

func loadType(m PackageMap) PackageType {
	typ := ""
	load := func(packageType PackageType) {
		if packageType == "" {
			return
		}

		if typ == "" {
			typ = packageType
			return
		} else {
			if typ != packageType {
				panic("inconsistent package type")
			}
		}
	}
	for _, p := range m {
		load(p.Type)
		load(loadType(p.SubPackages))
	}
	return typ
}

func (m *MainPackages) MergePseudoPacakge(modules []*debug.Module) {
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
		p.loaded = true
		p.Name = module.Path
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

		if p.loaded {
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

	// create pseudo package for types
	typs := make(PackageMap)
	for _, t := range []PackageType{
		PackageTypeStd,
		PackageTypeVendor,
		PackageTypeGenerated,
		PackageTypeUnknown,
		PackageTypeSelf} {
		typs[t] = &Package{
			Name:        t,
			Functions:   make([]*Function, 0),
			Type:        t,
			SubPackages: make(PackageMap),
			loaded:      true,
		}
	}
	for k, v := range newPackages {
		typ := v.Type
		if typ == "" {
			typ = loadType(v.SubPackages)
		}
		if typ == "" {
			panic("no type for package " + k)
		}

		typs[typ].SubPackages[k] = v
	}

	for k, v := range typs {
		if len(v.SubPackages) == 0 {
			delete(typs, k)
		}
	}

	m.Packages = typs
}

func (m *MainPackages) Add(gp *gore.Package, typ PackageType, pclntab *gosym.Table) {
	name, err := PrefixToPath(gp.Name)
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
			Pkg:      p,
		})
	}

	container[id] = p

	// also update the link
	m.link[gp.Name] = p
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
	grPkg  *gore.Package
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

	pkgs := NewMainPackages()
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
	pkgs.MergePseudoPacakge(modules)

	log.Println("Loading packages done")

	return nil
}
