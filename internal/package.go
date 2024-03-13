package internal

import (
	"debug/gosym"
	"github.com/goretk/gore"
	"log"
	"strings"
)

type PackageMap = map[string]*Package

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

func (m *MainPackages) Add(gp *gore.Package, typ PackageType, pclntab *gosym.Table) {
	parts := strings.Split(gp.Name, "/")
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
			panic("duplicate package " + gp.Name)
		}
		subs = c.SubPackages
	}
	p := &Package{
		Name:        Deduplicate(gp.Name),
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

type PackageType string

const (
	PackageTypeSelf      PackageType = "self"
	PackageTypeStd       PackageType = "std"
	PackageTypeVendor    PackageType = "vendor"
	PackageTypeGenerated PackageType = "generated"
	PackageTypeUnknown   PackageType = "unknown"
)

type Package struct {
	Name        string              `json:"name"`
	Functions   []*Function         `json:"functions"`
	Type        PackageType         `json:"type"`
	SubPackages map[string]*Package `json:"subPackages"`

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

	log.Println("Loading packages done")

	return nil
}
