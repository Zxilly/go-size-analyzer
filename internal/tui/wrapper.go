package tui

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/charmbracelet/bubbles/table"
	"github.com/dustin/go-humanize"
	"github.com/samber/lo"
	"path/filepath"
	"sync"
)

type wrappers []wrapper

func (w wrappers) ToRows() []table.Row {
	return lo.Map(w, func(item wrapper, _ int) table.Row {
		return item.toRow()
	})
}

// wrapper union type
type wrapper struct {
	pkg      *entity.Package
	section  *entity.Section
	file     *entity.File
	function *entity.Function

	childrenCache []wrapper
	cacheOnce     *sync.Once

	parent *wrapper
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

func buildPackageChildren(pkg *entity.Package) wrappers {
	ret := make([]wrapper, 0)
	for _, k := range pkg.Files {
		ret = append(ret, newWrapper(k))
	}

	for _, k := range utils.SortedKeys(pkg.SubPackages) {
		ret = append(ret, newWrapper(pkg.SubPackages[k]))
	}
	return ret
}

func (w wrapper) size() uint64 {
	switch {
	case w.pkg != nil:
		return w.pkg.Size
	case w.section != nil:
		return w.section.Size
	case w.file != nil:
		return w.file.FullSize()
	case w.function != nil:
		return w.function.Size()
	default:
		panic("invalid wrapper")
	}
}

func (w wrapper) typ() string {
	switch {
	case w.pkg != nil:
		return "Package"
	case w.section != nil:
		return "Section"
	case w.file != nil:
		return "File"
	case w.function != nil:
		return "Function"
	default:
		panic("invalid wrapper")
	}
}

func (w wrapper) toRow() table.Row {
	return table.Row{
		w.Title(),
		w.typ(),
		humanize.Bytes(w.size()),
	}
}

func (w wrapper) hasChildren() bool {
	return len(w.children()) > 0
}

func (w wrapper) children() wrappers {
	w.cacheOnce.Do(func() {
		var ret []wrapper
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
		for _, k := range ret {
			k.parent = &w
		}
		w.childrenCache = ret
	})

	return w.childrenCache
}
