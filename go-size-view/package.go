package go_size_view

import (
	"debug/gosym"
	"github.com/goretk/gore"
	"maps"
)

func extractPackages(file *gore.GoFile, k *KnownInfo) (*TypedPackages, error) {
	pkgs := new(TypedPackages)

	pkgs.nameToPkg = make(map[string]*Packages)

	pclntab, err := file.PCLNTab()
	if err != nil {
		return nil, err
	}

	self, err := file.GetPackages()
	if err != nil {
		return nil, err
	}
	selfPkgs, n, err := loadPackagesFromGorePackages(self, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Self = selfPkgs
	maps.Copy(pkgs.nameToPkg, n)

	grStd, _ := file.GetSTDLib()
	std, n, err := loadPackagesFromGorePackages(grStd, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Std = std
	maps.Copy(pkgs.nameToPkg, n)

	grVendor, _ := file.GetVendors()
	vendor, n, err := loadPackagesFromGorePackages(grVendor, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Vendor = vendor
	maps.Copy(pkgs.nameToPkg, n)

	grGenerated, _ := file.GetGeneratedPackages()
	generated, n, err := loadPackagesFromGorePackages(grGenerated, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Generated = generated
	maps.Copy(pkgs.nameToPkg, n)

	grUnknown, _ := file.GetUnknown()
	unknown, n, err := loadPackagesFromGorePackages(grUnknown, k, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Unknown = unknown
	maps.Copy(pkgs.nameToPkg, n)

	return pkgs, nil
}

func loadPackagesFromGorePackages(gr []*gore.Package, k *KnownInfo, pclntab *gosym.Table) ([]*Packages, map[string]*Packages, error) {
	pkgs := make([]*Packages, 0, len(gr))
	nameToPkg := make(map[string]*Packages)
	for _, g := range gr {
		pkg, err := loadPackageFromGore(g, k, pclntab)
		if err != nil {
			return nil, nil, err
		}
		pkgs = append(pkgs, pkg)
		nameToPkg[pkg.Name] = pkg
	}
	return pkgs, nameToPkg, nil
}

func loadPackageFromGore(pkg *gore.Package, k *KnownInfo, pclntab *gosym.Table) (*Packages, error) {
	files := map[string]*File{}

	getFile := func(path string) *File {
		f, ok := files[path]
		if !ok {
			nf := &File{
				Path:      path,
				Functions: make([]*gore.Function, 0),
			}
			files[path] = nf
			return nf
		}
		return f
	}

	setAddrMark := func(addr, size uint64) {
		k.FoundAddr.Insert(addr, size)
	}

	for _, m := range pkg.Methods {
		src, _, _ := pclntab.PCToLine(m.Offset)

		setAddrMark(m.Offset, m.End-m.Offset)

		mf := getFile(src)
		mf.Functions = append(mf.Functions, m.Function)
	}

	for _, f := range pkg.Functions {
		src, _, _ := pclntab.PCToLine(f.Offset)

		setAddrMark(f.Offset, f.End-f.Offset)

		ff := getFile(src)
		ff.Functions = append(ff.Functions, f)
	}

	filesSlice := make([]*File, 0, len(files))
	for _, f := range files {
		filesSlice = append(filesSlice, f)
	}

	return &Packages{
		Name:  pkg.Name,
		Files: filesSlice,
		grPkg: pkg,
	}, nil
}

type TypedPackages struct {
	Self      []*Packages
	Std       []*Packages
	Vendor    []*Packages
	Generated []*Packages
	Unknown   []*Packages

	nameToPkg map[string]*Packages
}

type Packages struct {
	Name     string
	Files    []*File
	Sections map[string]uint64
	grPkg    *gore.Package
}

func (p *Packages) GetSize() uint64 {
	var size uint64 = 0
	for _, f := range p.Files {
		size += f.GetSize()
	}
	for _, s := range p.Sections {
		size += s
	}
	return size
}
