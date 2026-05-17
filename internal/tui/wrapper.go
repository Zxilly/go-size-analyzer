package tui

import (
	"cmp"
	"fmt"
	"html"
	"slices"
	"strings"
	"sync"
	"unicode"

	"charm.land/bubbles/v2/table"
	"github.com/dustin/go-humanize"
	"github.com/samber/lo"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

const (
	unknownSourceLabel = "(unknown source)"
	rootSourceLabel    = "(root)"
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

	descCache string
	descOnce  *sync.Once

	parent *wrapper
}

func newWrapper(cnt any) wrapper {
	w := wrapper{cacheOnce: &sync.Once{}, descOnce: &sync.Once{}}
	switch v := cnt.(type) {
	case *entity.Package:
		w.pkg = v
	case *entity.Section:
		w.section = v
	case *entity.File:
		w.file = v
	case *entity.Function:
		w.function = v
	default:
		panic("invalid wrapper")
	}
	return w
}

func functionDisplayName(f *entity.Function) string {
	switch f.Type {
	case entity.FuncTypeFunction:
		return f.Name
	case entity.FuncTypeMethod:
		return fmt.Sprintf("%s.%s", f.Receiver, f.Name)
	default:
		panic("invalid function type")
	}
}

func markdownText(s string) string {
	return html.EscapeString(s)
}

func subPackageDisplayName(name string) string {
	name = strings.TrimLeft(name, `/\`)
	if name == "" {
		return unknownSourceLabel
	}
	return name
}

func fileDisplayPath(f *entity.File) string {
	path := cleanSourcePath(f.FilePath)
	if path == "" {
		return unknownSourceLabel
	}
	return path
}

func cleanSourcePath(path string) string {
	return strings.TrimFunc(path, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	})
}

func splitFilePath(path string) (dir string, name string) {
	path = cleanSourcePath(path)
	if path == "" {
		return unknownSourceLabel, unknownSourceLabel
	}
	i := strings.LastIndexAny(path, `/\`)
	if i < 0 {
		return rootSourceLabel, path
	}
	dir = path[:i]
	if dir == "" {
		dir = rootSourceLabel
	}
	name = cleanSourcePath(path[i+1:])
	if name == "" {
		name = unknownSourceLabel
	}
	return dir, name
}

func fileDisplayName(f *entity.File) string {
	_, name := splitFilePath(f.FilePath)
	return name
}

func writeFileGroups(writeln func(string, ...any), files []*entity.File) {
	filesByDir := make(map[string][]string)
	for _, f := range files {
		dir, name := splitFilePath(f.FilePath)
		filesByDir[dir] = append(filesByDir[dir], name)
	}

	dirs := utils.SortedKeys(filesByDir)
	for _, dir := range dirs {
		writeln("### %s", markdownText(dir))
		slices.Sort(filesByDir[dir])
		for _, name := range filesByDir[dir] {
			writeln("- %s", markdownText(name))
		}
	}
}

func (w *wrapper) Title() string {
	switch {
	case w.pkg != nil:
		return w.pkg.Name
	case w.section != nil:
		return w.section.Name
	case w.file != nil:
		return fileDisplayName(w.file)
	case w.function != nil:
		return functionDisplayName(w.function)
	default:
		panic("invalid wrapper")
	}
}

func sizeLine(label string, size uint64) string {
	return fmt.Sprintf("- **%s:** %s (%d Bytes)", label, humanize.Bytes(size), size)
}

// Description is hot — called on every left-table selection change. The
// output is deterministic per wrapper, so memoize via sync.Once.
func (w *wrapper) Description() string {
	w.descOnce.Do(func() {
		w.descCache = w.describe()
	})
	return w.descCache
}

func (w *wrapper) describe() string {
	sb := new(strings.Builder)

	writeln := func(format string, args ...any) {
		_, _ = fmt.Fprintf(sb, format+"\n", args...)
	}

	switch {
	case w.pkg != nil:
		writeln("# %s _(Package)_", markdownText(w.pkg.Name))
		writeln("")
		writeln(sizeLine("Size", w.pkg.Size))
		writeln("- **Type:** %s", markdownText(w.pkg.Type))

		if len(w.pkg.ImportedBy) > 0 {
			writeln("")
			writeln("## Imported By")
			writeln("")
			for _, k := range w.pkg.ImportedBy {
				writeln("- %s", markdownText(k))
			}
		}
		if len(w.pkg.Files) > 0 {
			writeln("")
			writeln("## Files")
			writeFileGroups(writeln, w.pkg.Files)
		}
		if len(w.pkg.SubPackages) > 0 {
			writeln("")
			writeln("## SubPackages")
			writeln("")
			for _, k := range utils.SortedKeys(w.pkg.SubPackages) {
				writeln("- %s", markdownText(subPackageDisplayName(k)))
			}
		}
		if len(w.pkg.Symbols) > 0 {
			writeln("")
			writeln("## Symbols")
			writeln("")
			syms := slices.Clone(w.pkg.Symbols)
			slices.SortFunc(syms, func(a, b *entity.Symbol) int {
				return -cmp.Compare(a.Size, b.Size)
			})
			for _, s := range syms {
				writeln("- `%s` — %s", s.Name, humanize.Bytes(s.Size))
			}
		}
	case w.section != nil:
		writeln("# %s _(Section)_", markdownText(w.section.Name))
		writeln("")
		writeln(sizeLine("Size", w.section.Size))
		writeln(sizeLine("File Size", w.section.FileSize))
		writeln(sizeLine("Known Size", w.section.KnownSize))
		if w.section.Addr != 0 || w.section.AddrEnd != 0 {
			writeln("- **Addr:** `0x%x - 0x%x`", w.section.Addr, w.section.AddrEnd)
		}
		writeln("- **Offset:** `0x%x - 0x%x`", w.section.Offset, w.section.Offset+w.section.FileSize)
		writeln("- **Only In Memory:** %t", w.section.OnlyInMemory)
	case w.file != nil:
		writeln("# %s _(File)_", markdownText(fileDisplayName(w.file)))
		writeln("")
		writeln("- **Path:** %s", markdownText(fileDisplayPath(w.file)))
		writeln(sizeLine("Size", w.file.FullSize()))
		writeln("- **Package:** %s", markdownText(w.file.PkgName))
		if len(w.file.Functions) > 0 {
			writeln("")
			writeln("## Functions")
			writeln("")
			funcs := slices.Clone(w.file.Functions)
			slices.SortFunc(funcs, func(a, b *entity.Function) int {
				return -cmp.Compare(a.Size(), b.Size())
			})
			for _, k := range funcs {
				writeln("- `%s` — %s", functionDisplayName(k), humanize.Bytes(k.Size()))
			}
		}
	case w.function != nil:
		writeln("# %s _(Function)_", markdownText(functionDisplayName(w.function)))
		writeln("")
		writeln(sizeLine("Size", w.function.Size()))
		writeln("- **Type:** %s", markdownText(w.function.Type))
		if w.function.Type == entity.FuncTypeMethod {
			writeln("- **Receiver:** %s", markdownText(w.function.Receiver))
		}
		writeln(sizeLine("Code Size", w.function.CodeSize))
		writeln(sizeLine("Pcln Size", w.function.PclnSize.Size()))
		writeln("- **Addr:** `0x%x - 0x%x`", w.function.Addr, w.function.Addr+w.function.CodeSize)

		writeln("")
		writeln("## Pcln Details")
		writeln("")
		writeln("- **Func Name:** %d Bytes", w.function.PclnSize.Name)
		writeln("- **File Name Tab:** %d Bytes", w.function.PclnSize.PCFile)
		writeln("- **PC to Stack Pointer Table:** %d Bytes", w.function.PclnSize.PCSP)
		writeln("- **PC to Line Number Table:** %d Bytes", w.function.PclnSize.PCLN)
		writeln("- **Header:** %d Bytes", w.function.PclnSize.Header)
		writeln("- **Func Data:** %d Bytes", w.function.PclnSize.FuncData)

		if len(w.function.PclnSize.PCData) > 0 {
			writeln("")
			writeln("### PC Data")
			writeln("")
			for _, k := range utils.SortedKeys(w.function.PclnSize.PCData) {
				writeln("- **%s:** %d Bytes", k, w.function.PclnSize.PCData[k])
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
