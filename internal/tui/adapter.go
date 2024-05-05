package tui

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/charmbracelet/bubbles/list"
	"github.com/samber/lo"
	"path/filepath"
)

var _ list.Item = wrapper{}

// wrapper union type
type wrapper struct {
	pkg      *entity.Package
	section  *entity.Section
	file     *entity.File
	function *entity.Function

	parent *wrapper
}

func (w wrapper) FilterValue() string {
	return w.Title()
}

func newWrapper(cnt any) wrapper {
	switch v := cnt.(type) {
	case *entity.Package:
		return wrapper{pkg: v}
	case *entity.Section:
		return wrapper{section: v}
	case *entity.File:
		return wrapper{file: v}
	case *entity.Function:
		return wrapper{function: v}
	default:
		panic("invalid wrapper")
	}
}

func (w wrapper) Title() string {
	switch {
	case w.pkg != nil:
		return w.pkg.Name
	case w.section != nil:
		return w.section.Name
	case w.file != nil:
		return filepath.Base(w.file.FilePath)
	case w.function != nil:
		return w.function.Name
	default:
		panic("invalid wrapper")
	}
}

func (w wrapper) Description() string {
	// fixme: implement me
	return "not implemented yet"
}

func buildPackageChildren(pkg *entity.Package) []wrapper {
	ret := make([]wrapper, 0)
	for _, k := range pkg.Files {
		ret = append(ret, newWrapper(k))
	}

	for _, k := range utils.SortedKeys(pkg.SubPackages) {
		ret = append(ret, newWrapper(pkg.SubPackages[k]))
	}
	return ret
}

func (w wrapper) children() (ret []wrapper) {
	defer func() {
		for _, k := range ret {
			k.parent = &w
		}
	}()

	switch {
	case w.pkg != nil:
		ret = buildPackageChildren(w.pkg)
	case w.section != nil || w.function != nil:
		// no children for section and function
		ret = make([]wrapper, 0)
	case w.file != nil:
		ret = lo.Map(w.file.Functions, func(item *entity.Function, _ int) wrapper {
			return newWrapper(item)
		})
	default:
		panic("invalid wrapper")
	}
	return
}
