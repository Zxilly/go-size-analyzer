package go_size_view

import (
	"debug/gosym"
	"github.com/goretk/gore"
)

func configurePackages(file *gore.GoFile, sections *SectionMap) (*TypedPackages, error) {
	pkgs := new(TypedPackages)
	pclntab, err := file.PCLNTab()
	if err != nil {
		return nil, err
	}

	self, err := file.GetPackages()
	if err != nil {
		return nil, err
	}
	selfPkgs, err := loadPackagesFromGorePackages(self, sections, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Self = selfPkgs

	grStd, _ := file.GetSTDLib()
	std, err := loadPackagesFromGorePackages(grStd, sections, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Std = std

	grVendor, _ := file.GetVendors()
	vendor, err := loadPackagesFromGorePackages(grVendor, sections, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Vendor = vendor

	grGenerated, _ := file.GetGeneratedPackages()
	generated, err := loadPackagesFromGorePackages(grGenerated, sections, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Generated = generated

	grUnknown, _ := file.GetUnknown()
	unknown, err := loadPackagesFromGorePackages(grUnknown, sections, pclntab)
	if err != nil {
		return nil, err
	}
	pkgs.Unknown = unknown

	return pkgs, nil
}

func loadPackagesFromGorePackages(gr []*gore.Package, sections *SectionMap, pclntab *gosym.Table) ([]*Packages, error) {
	pkgs := make([]*Packages, 0, len(gr))
	for _, g := range gr {
		pkg, err := loadPackageFromGore(g, sections, pclntab)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func loadPackageFromGore(pkg *gore.Package, sections *SectionMap, pclntab *gosym.Table) (*Packages, error) {
	size := uint64(0)
	files := map[string]*File{}

	getFile := func(path string) *File {
		f, ok := files[path]
		if !ok {
			nf := &File{
				Size:      0,
				Path:      path,
				Functions: make([]*gore.Function, 0),
			}
			files[path] = nf
			return nf
		}
		return f
	}

	setSymbolMark := func(offset uint64) {
		sym := sections.SymTab.Symbols[offset]
		if sym != nil {
			sym.SizeCounted = true
		}
	}

	for _, m := range pkg.Methods {
		src, _, _ := pclntab.PCToLine(m.Offset)

		size += m.End - m.Offset
		err := sections.IncreaseKnown(m.Offset, m.End)
		if err != nil {
			println(m.PackageName, m.Name, "no section found", m.Offset, m.End)
			//return nil, err
		}

		setSymbolMark(m.Offset)

		mf := getFile(src)
		mf.Functions = append(mf.Functions, m.Function)
		mf.Size += m.End - m.Offset
	}

	for _, f := range pkg.Functions {
		src, _, _ := pclntab.PCToLine(f.Offset)

		size += f.End - f.Offset
		err := sections.IncreaseKnown(f.Offset, f.End)
		if err != nil {
			println(f.PackageName, f.Name, "no section found", f.Offset, f.End)
			//return nil, err
		}

		setSymbolMark(f.Offset)

		ff := getFile(src)
		ff.Functions = append(ff.Functions, f)
		ff.Size += f.End - f.Offset
	}

	filesSlice := make([]*File, 0, len(files))
	for _, f := range files {
		filesSlice = append(filesSlice, f)
	}

	return &Packages{
		Name:  pkg.Name,
		Size:  size,
		Files: filesSlice,
		grPkg: pkg,
	}, nil
}
