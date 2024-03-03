package pkg

import (
	"debug/gosym"
	"github.com/goretk/gore"
	"log"
	"maps"
)

func (k *KnownInfo) LoadPackages(file *gore.GoFile) error {
	log.Println("Loading packages...")

	pkgs := new(TypedPackages)

	pkgs.NameToPkg = make(map[string]*Package)

	pclntab, err := file.PCLNTab()
	if err != nil {
		return err
	}

	self, err := file.GetPackages()
	if err != nil {
		return err
	}
	selfPkgs, n, err := loadGorePackages(self, k, pclntab)
	if err != nil {
		return err
	}
	pkgs.Self = selfPkgs
	maps.Copy(pkgs.NameToPkg, n)

	grStd, _ := file.GetSTDLib()
	std, n, err := loadGorePackages(grStd, k, pclntab)
	if err != nil {
		return err
	}
	pkgs.Std = std
	maps.Copy(pkgs.NameToPkg, n)

	grVendor, _ := file.GetVendors()
	vendor, n, err := loadGorePackages(grVendor, k, pclntab)
	if err != nil {
		return err
	}
	pkgs.Vendor = vendor
	maps.Copy(pkgs.NameToPkg, n)

	grGenerated, _ := file.GetGeneratedPackages()
	generated, n, err := loadGorePackages(grGenerated, k, pclntab)
	if err != nil {
		return err
	}
	pkgs.Generated = generated
	maps.Copy(pkgs.NameToPkg, n)

	grUnknown, _ := file.GetUnknown()
	unknown, n, err := loadGorePackages(grUnknown, k, pclntab)
	if err != nil {
		return err
	}
	pkgs.Unknown = unknown
	maps.Copy(pkgs.NameToPkg, n)

	log.Println("Loading packages done")

	k.Packages = pkgs

	return nil
}

func loadGorePackages(gr []*gore.Package, k *KnownInfo, pclntab *gosym.Table) ([]*Package, map[string]*Package, error) {
	pkgs := make([]*Package, 0, len(gr))
	nameToPkg := make(map[string]*Package)
	for _, g := range gr {
		pkg, err := loadGorePackage(g, k, pclntab)
		if err != nil {
			return nil, nil, err
		}
		pkgs = append(pkgs, pkg)
		nameToPkg[pkg.Name] = pkg
	}
	return pkgs, nameToPkg, nil
}

func loadGorePackage(pkg *gore.Package, k *KnownInfo, pclntab *gosym.Table) (*Package, error) {
	p := &Package{
		Name:      pkg.Name,
		Methods:   pkg.Methods,
		Functions: pkg.Functions,
	}

	setAddrMark := func(addr, size uint64, meta GoPclntabMeta) {
		// everything in the pclntab is text
		k.KnownAddr.InsertPclntab(addr, size, p, meta)
	}

	for _, m := range pkg.Methods {
		src, _, _ := pclntab.PCToLine(m.Offset)

		setAddrMark(m.Offset, m.End-m.Offset, GoPclntabMeta{
			FuncName:    Deduplicate(m.Name),
			PackageName: Deduplicate(m.PackageName),
			Type:        FuncTypeMethod,
			Receiver:    Deduplicate(m.Receiver),
			Filepath:    Deduplicate(src),
		})
	}

	for _, f := range pkg.Functions {
		src, _, _ := pclntab.PCToLine(f.Offset)

		setAddrMark(f.Offset, f.End-f.Offset, GoPclntabMeta{
			FuncName:    Deduplicate(f.Name),
			PackageName: Deduplicate(f.PackageName),
			Type:        FuncTypeFunction,
			Receiver:    Deduplicate(""),
			Filepath:    Deduplicate(src),
		})
	}

	return p, nil
}

type TypedPackages struct {
	Self      []*Package
	Std       []*Package
	Vendor    []*Package
	Generated []*Package
	Unknown   []*Package

	NameToPkg map[string]*Package // available after LoadPackages, loadPackagesFromGorePackage[s]
}

func (tp *TypedPackages) GetPackages() []*Package {
	var pkgs []*Package

	// use this order to make it more likely to set the disasm result to the right package
	pkgs = append(pkgs, tp.Unknown...)
	pkgs = append(pkgs, tp.Generated...)
	pkgs = append(pkgs, tp.Std...)
	pkgs = append(pkgs, tp.Vendor...)
	pkgs = append(pkgs, tp.Self...)
	return pkgs
}

func (tp *TypedPackages) GetPackageAndCountFn() ([]*Package, int) {
	pkgs := tp.GetPackages()
	cnt := 0
	for _, p := range pkgs {
		cnt += len(p.GetFunctions())
		cnt += len(p.GetMethods())
	}
	return pkgs, cnt
}

type Package struct {
	Name      string
	Functions []*gore.Function
	Methods   []*gore.Method
	grPkg     *gore.Package
}

func (p *Package) GetFunctions() []*gore.Function {
	return p.Functions
}

func (p *Package) GetMethods() []*gore.Method {
	return p.Methods
}
