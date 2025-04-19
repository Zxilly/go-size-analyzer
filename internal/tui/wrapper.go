package tui

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/table"
	"github.com/dustin/go-humanize"
	"github.com/samber/lo"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
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

	writeln := func(format string, args ...any) {
		_, _ = fmt.Fprintf(sb, format+"\n", args...)
	}

	switch {
	case w.pkg != nil:
		{
			writeln("Package: %s", w.pkg.Name)
			writeln("Size: %s (%d Bytes)", humanize.Bytes(w.pkg.Size), w.pkg.Size)
			writeln("Type: %s", w.pkg.Type)

			if len(w.pkg.ImportedBy) > 0 {
				writeln("")
				writeln("Imported By:")
				for _, k := range w.pkg.ImportedBy {
					writeln("  %s", k)
				}
			}

			if len(w.pkg.Files) > 0 {
				writeln("")
				writeln("Files:")
				for _, k := range w.pkg.Files {
					writeln("  %s", k.FilePath)
				}
			}
			if len(w.pkg.SubPackages) > 0 {
				writeln("")
				writeln("SubPackages:")
				for _, k := range utils.SortedKeys(w.pkg.SubPackages) {
					writeln("  %s", k)
				}
			}
			if len(w.pkg.Symbols) > 0 {
				writeln("")
				writeln("Symbols:")
				for _, k := range w.pkg.Symbols {
					writeln("  %s", k)
				}
			}
		}
	case w.section != nil:
		{
			writeln("Section: %s", w.section.Name)
			writeln("Size: %s (%d Bytes)", humanize.Bytes(w.section.Size), w.section.Size)
			writeln("File Size: %s (%d Bytes)",
				humanize.Bytes(w.section.FileSize), w.section.FileSize)
			writeln("Known Size: %s (%d Bytes)",
				humanize.Bytes(w.section.KnownSize), w.section.KnownSize)
			writeln("Addr: 0x%x - 0x%x", w.section.Addr, w.section.Addr+w.section.Size)
			writeln("Offset: 0x%x - 0x%x", w.section.Offset, w.section.Offset+w.section.FileSize)
			writeln("Only In Memory: %t", w.section.OnlyInMemory)
		}
	case w.file != nil:
		{
			writeln("File: %s", w.file.FilePath)
			writeln("Size: %s (%d Bytes)",
				humanize.Bytes(w.file.FullSize()), w.file.FullSize())
			writeln("Package: %s", w.file.PkgName)
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

					writeln("  %s", name)
				}
			}
		}
	case w.function != nil:
		{
			writeln("Function: %s", w.function.Name)
			writeln("Size: %s (%d Bytes)",
				humanize.Bytes(w.function.Size()), w.function.Size())
			writeln("Type: %s", w.function.Type)
			if w.function.Type == entity.FuncTypeMethod {
				writeln("Receiver: %s", w.function.Receiver)
			}

			writeln("Code Size: %s (%d Bytes)",
				humanize.Bytes(w.function.CodeSize), w.function.CodeSize)
			writeln("Pcln Size: %s (%d Bytes)",
				humanize.Bytes(w.function.PclnSize.Size()), w.function.PclnSize.Size())
			writeln("Addr: 0x%x - 0x%x", w.function.Addr, w.function.Addr+w.function.CodeSize)

			writeln("")
			writeln("Pcln Details:")
			writeln("  Func Name: %d Bytes", w.function.PclnSize.Name)
			writeln("  File Name Tab: %d Bytes", w.function.PclnSize.PCFile)
			writeln("  PC to Stack Pointer Table: %d Bytes", w.function.PclnSize.PCSP)
			writeln("  PC to Line Number Table: %d Bytes", w.function.PclnSize.PCLN)
			writeln("  Header: %d Bytes", w.function.PclnSize.Header)
			writeln("  Func Data: %d Bytes", w.function.PclnSize.FuncData)
			writeln("  PC Data:")
			for _, k := range utils.SortedKeys(w.function.PclnSize.PCData) {
				writeln("    %s: %d Bytes", k, w.function.PclnSize.PCData[k])
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
		return w.section.FileSize - w.section.KnownSize
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
