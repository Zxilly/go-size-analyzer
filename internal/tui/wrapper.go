package tui

import (
	"cmp"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/charmbracelet/bubbles/table"
	"github.com/dustin/go-humanize"
	"github.com/samber/lo"
	"path/filepath"
	"slices"
	"strings"
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
		return wrapper{pkg: v, cacheOnce: &sync.Once{}}
	case *entity.Section:
		return wrapper{section: v, cacheOnce: &sync.Once{}}
	case *entity.File:
		return wrapper{file: v, cacheOnce: &sync.Once{}}
	case *entity.Function:
		return wrapper{function: v, cacheOnce: &sync.Once{}}
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
		switch w.function.Type {
		case entity.FuncTypeFunction:
			return w.function.Name
		case entity.FuncTypeMethod:
			return fmt.Sprintf("%s.%s", w.function.Receiver, w.function.Name)
		default:
			panic("invalid function type")
		}
	default:
		panic("invalid wrapper")
	}
}

func (w *wrapper) Description() string {
	sb := new(strings.Builder)

	writeln := func(s string) {
		sb.WriteString(s)
		sb.WriteRune('\n')
	}

	switch {
	case w.pkg != nil:
		{
			writeln(fmt.Sprintf("Package: %s", w.pkg.Name))
			writeln(fmt.Sprintf("Size: %s (%d Bytes)", humanize.Bytes(w.pkg.Size), w.pkg.Size))
			writeln(fmt.Sprintf("Type: %s", w.pkg.Type))

			if len(w.pkg.Files) > 0 {
				writeln("")
				writeln("Files:")
				for _, k := range w.pkg.Files {
					writeln(fmt.Sprintf("  %s", k.FilePath))
				}
			}
			if len(w.pkg.SubPackages) > 0 {
				writeln("")
				writeln("SubPackages:")
				for _, k := range utils.SortedKeys(w.pkg.SubPackages) {
					writeln(fmt.Sprintf("  %s", k))
				}
			}
			if len(w.pkg.Symbols) > 0 {
				writeln("")
				writeln("Symbols:")
				for _, k := range w.pkg.Symbols {
					writeln(fmt.Sprintf("  %s", k))
				}
			}
		}
	case w.section != nil:
		{
			writeln(fmt.Sprintf("Section: %s", w.section.Name))
			writeln(fmt.Sprintf("Size: %s (%d Bytes)", humanize.Bytes(w.section.Size), w.section.Size))
			writeln(fmt.Sprintf("File Size: %s (%d Bytes)", humanize.Bytes(w.section.FileSize), w.section.FileSize))
			writeln(fmt.Sprintf("Addr: 0x%x - 0x%x", w.section.Addr, w.section.Addr+w.section.Size))
			writeln(fmt.Sprintf("Offset: 0x%x - 0x%x", w.section.Offset, w.section.Offset+w.section.FileSize))
		}
	case w.file != nil:
		{
			writeln(fmt.Sprintf("File: %s", w.file.FilePath))
			writeln(fmt.Sprintf("Size: %s (%d Bytes)", humanize.Bytes(w.file.FullSize()), w.file.FullSize()))
			writeln(fmt.Sprintf("Package: %s", w.file.Pkg.Name))
			if len(w.file.Functions) > 0 {
				writeln("")
				writeln("Functions:")
				for _, k := range w.file.Functions {
					var name string
					switch k.Type {
					case entity.FuncTypeFunction:
						name = k.Name
					case entity.FuncTypeMethod:
						name = fmt.Sprintf("%s.%s", k.Receiver, k.Name)
					}

					writeln(fmt.Sprintf("  %s", name))
				}
			}
		}
	case w.function != nil:
		{
			writeln(fmt.Sprintf("Function: %s", w.function.Name))
			writeln(fmt.Sprintf("Size: %s (%d Bytes)", humanize.Bytes(w.function.Size()), w.function.Size()))
			writeln(fmt.Sprintf("Type: %s", w.function.Type))
			if w.function.Type == entity.FuncTypeMethod {
				writeln(fmt.Sprintf("Receiver: %s", w.function.Receiver))
			}

			writeln(fmt.Sprintf("Code Size: %s (%d Bytes)", humanize.Bytes(w.function.CodeSize), w.function.CodeSize))
			writeln(fmt.Sprintf("Pcln Size: %s (%d Bytes)", humanize.Bytes(w.function.PclnSize.Size()), w.function.PclnSize.Size()))
			writeln(fmt.Sprintf("Addr: 0x%x - 0x%x", w.function.Addr, w.function.Addr+w.function.CodeSize))

			writeln("")
			writeln("Pcln Details:")
			writeln(fmt.Sprintf("  Func Name: %d Bytes", w.function.PclnSize.Name))
			writeln(fmt.Sprintf("  File Name Tab: %d Bytes", w.function.PclnSize.PCFile))
			writeln(fmt.Sprintf("  PC to Stack Pointer Table: %d Bytes", w.function.PclnSize.PCSP))
			writeln(fmt.Sprintf("  PC to Line Number Table: %d Bytes", w.function.PclnSize.PCLN))
			writeln(fmt.Sprintf("  Header: %d Bytes", w.function.PclnSize.Header))
			writeln(fmt.Sprintf("  Func Data: %d Bytes", w.function.PclnSize.FuncData))
			writeln("  PC Data:")
			for _, k := range utils.SortedKeys(w.function.PclnSize.PCData) {
				writeln(fmt.Sprintf("    %s: %d Bytes", k, w.function.PclnSize.PCData[k]))
			}
		}
	default:
		panic("unreachable")
	}

	return sb.String()
}

func sortWrappers(wrappers wrappers) {
	slices.SortFunc(wrappers, func(a, b wrapper) int {
		return -cmp.Compare(a.size(), b.size())
	})
}

func buildPackageChildren(pkg *entity.Package) wrappers {
	subs := make([]wrapper, 0)
	for _, k := range utils.SortedKeys(pkg.SubPackages) {
		subs = append(subs, newWrapper(pkg.SubPackages[k]))
	}
	sortWrappers(subs)

	files := make([]wrapper, 0)
	for _, k := range pkg.Files {
		files = append(files, newWrapper(k))
	}
	sortWrappers(files)

	ret := make([]wrapper, 0, len(files)+len(subs))
	ret = append(ret, subs...)
	ret = append(ret, files...)

	return ret
}

func (w *wrapper) size() uint64 {
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

func (w *wrapper) toRow() table.Row {
	return table.Row{
		w.Title(),
		humanize.Bytes(w.size()),
	}
}

func (w *wrapper) hasChildren() bool {
	return len(w.children()) > 0
}

func (w *wrapper) children() wrappers {
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
			sortWrappers(ret)
		default:
			panic("invalid wrapper")
		}

		w.childrenCache = lo.Map(ret, func(item wrapper, _ int) wrapper {
			item.parent = w
			return item
		})
	})

	return w.childrenCache
}
