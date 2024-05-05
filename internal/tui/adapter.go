package tui

import (
	"cmp"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/samber/lo"
	"path/filepath"
	"slices"
)

// wrapper union type
type wrapper struct {
	pkg      *entity.Package
	section  *entity.Section
	file     *entity.File
	function *entity.Function
}

func newWrapper(cnt any) *wrapper {
	switch v := cnt.(type) {
	case *entity.Package:
		return &wrapper{pkg: v}
	case *entity.Section:
		return &wrapper{section: v}
	case *entity.File:
		return &wrapper{file: v}
	case *entity.Function:
		return &wrapper{function: v}
	default:
		panic("invalid wrapper")
	}
}

func (w *wrapper) Title() string {
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

func (w *wrapper) Description() string {
	// fixme: implement me
	return "not implemented yet"
}

func (w *wrapper) Children() []*wrapper {
	switch {
	case w.pkg != nil:
		{
			ret := make([]*wrapper, 0)
			for _, k := range w.pkg.Files {
				ret = append(ret, newWrapper(k))
			}
			slices.SortFunc(ret, func(a, b *wrapper) int {
				return cmp.Compare(a.file.FilePath, b.file.FilePath)
			})

			for _, k := range utils.SortedKeys(w.pkg.SubPackages) {
				ret = append(ret, newWrapper(w.pkg.SubPackages[k]))
			}
			return ret
		}
	case w.section != nil || w.function != nil:
		// no children for section and function
		return make([]*wrapper, 0)
	case w.file != nil:
		return lo.Map(w.file.Functions, func(item *entity.Function, _ int) *wrapper {
			return newWrapper(item)
		})
	default:
		panic("invalid wrapper")
	}
}
