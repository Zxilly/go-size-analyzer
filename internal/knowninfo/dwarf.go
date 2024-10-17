package knowninfo

import (
	"context"
	"debug/dwarf"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/ZxillyFork/gore"
	"github.com/ZxillyFork/gosym"
	"github.com/go-delve/delve/pkg/dwarf/op"

	dwarfutil "github.com/Zxilly/go-size-analyzer/internal/dwarf"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func safeGetEntryVal[T any](entry *dwarf.Entry, attr dwarf.Attr, name string) (T, bool) {
	v, ok := entry.Val(attr).(T)
	if !ok {
		slog.Debug(fmt.Sprintf("Failed to load DWARF %s: %s", name, dwarfutil.EntryPrettyPrint(entry)))
		return *new(T), false
	}
	return v, true
}

func (k *KnownInfo) AddDwarfVariable(entry *dwarf.Entry, d *dwarf.Data, pkg *entity.Package, ptrSize int) {
	insts, ok := safeGetEntryVal[[]byte](entry, dwarf.AttrLocation, "location attribute")
	if !ok {
		return
	}

	addr, _, err := op.ExecuteStackProgram(op.DwarfRegisters{}, insts, ptrSize, nil)
	if err != nil {
		level := slog.LevelDebug
		if !errors.Is(err, op.ErrMemoryReadUnavailable) {
			level = slog.LevelWarn
		}
		slog.Log(context.Background(),
			level,
			fmt.Sprintf(
				"Failed to execute location attribute for %s: %v",
				dwarfutil.EntryPrettyPrint(entry), err,
			),
		)
		return
	}

	contents, typSize, err := dwarfutil.SizeForDWARFVar(d, entry, uint64(addr), k.Wrapper.ReadAddr)
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to load DWARF var %s: %v", dwarfutil.EntryPrettyPrint(entry), err))
		return
	}

	entryName, ok := safeGetEntryVal[string](entry, dwarf.AttrName, "variable name")
	if !ok {
		slog.Debug(fmt.Sprintf("Failed to load DWARF var name: %s", dwarfutil.EntryPrettyPrint(entry)))
		return
	}

	symbol := entity.NewSymbol(entryName, uint64(addr), typSize, entity.AddrTypeData)

	ap := k.KnownAddr.InsertSymbolFromDWARF(symbol, pkg)
	if ap == nil {
		return
	}
	pkg.AddSymbol(symbol, ap)

	if len(contents) > 0 {
		for _, content := range contents {
			if content.Size == 0 {
				slog.Debug(fmt.Sprintf("zero size for %s", entryName))
				continue
			}

			if content.Addr == 0 {
				slog.Debug(fmt.Sprintf("zero addr for %s", entryName))
				continue
			}

			valueName := utils.Deduplicate(fmt.Sprintf("%s.%s", entryName, content.Name))

			symbol = entity.NewSymbol(valueName, content.Addr, content.Size, entity.AddrTypeData)

			ap = k.KnownAddr.InsertSymbolFromDWARF(symbol, pkg)
			if ap == nil {
				continue
			}
			pkg.AddSymbol(symbol, ap)
		}
	}
}

func (k *KnownInfo) AddDwarfSubProgram(
	isGo bool,
	d *dwarf.Data,
	subEntry *dwarf.Entry,
	pkg *entity.Package,
	readFileName func(entry *dwarf.Entry) string,
) {
	subEntryName, ok := safeGetEntryVal[string](subEntry, dwarf.AttrName, "function name")
	if !ok {
		return
	}

	ranges, err := d.Ranges(subEntry)
	if err != nil {
		slog.Debug(fmt.Sprintf("Failed to load DWARF function size: %v", err))
		return
	}

	if len(ranges) == 0 {
		// fixme: maybe compiler optimize it?
		// example: sqlite3 simpleDestroy
		slog.Debug(fmt.Sprintf("Failed to load DWARF function size, no range: %s", subEntryName))
		return
	}

	addr := ranges[0][0]
	size := ranges[0][1] - ranges[0][0]

	typ := entity.FuncTypeFunction
	receiverName := ""
	if isGo {
		receiverName = (&gosym.Sym{Name: subEntryName}).ReceiverName()
		if receiverName != "" {
			typ = entity.FuncTypeMethod
		}
	}

	filename := readFileName(subEntry)

	fn := &entity.Function{
		Name:     subEntryName,
		Addr:     addr,
		CodeSize: size,
		Type:     typ,
		Receiver: receiverName,
		PclnSize: entity.NewEmptyPclnSymbolSize(),
	}
	fn.Init()

	added := pkg.AddFuncIfNotExists(filename, fn)

	if added {
		k.KnownAddr.InsertTextFromDWARF(addr, size, fn)
	}
}

func (k *KnownInfo) GetPackageFromDwarfCompileUnit(cuEntry *dwarf.Entry) *entity.Package {
	cuLang, ok := safeGetEntryVal[int64](cuEntry, dwarf.AttrLanguage, "compile unit language")
	if !ok {
		return nil
	}

	cuName, ok := safeGetEntryVal[string](cuEntry, dwarf.AttrName, "compile unit name")
	if !ok {
		return nil
	}

	var pkg *entity.Package

	if cuLang == dwarfutil.DwLangGo {
		// if we have load it with pclntab?
		pkg = k.Deps.Trie.Get(cuName)
		if pkg == nil {
			pkg = entity.NewPackage()
			pkg.Name = cuName
		}
		typ := entity.PackageTypeVendor
		if cuName == "main" {
			typ = entity.PackageTypeMain
		} else if gore.IsStandardLibrary(cuName) {
			typ = entity.PackageTypeStd
		}
		pkg.Type = typ
	} else {
		pkgName := fmt.Sprintf("CGO %s", dwarfutil.LanguageString(cuLang))
		pkg = k.Deps.Trie.Get(pkgName)
		if pkg == nil {
			pkg = entity.NewPackage()
			pkg.Name = pkgName
			pkg.Type = entity.PackageTypeCGO
			k.Deps.Trie.Put(pkgName, pkg)
		}
	}

	return pkg
}

type EntryFeeder func(e *dwarf.Entry)

func (k *KnownInfo) GetDwarfCompileUnitFeeder(d *dwarf.Data, cuEntry *dwarf.Entry, ptrSize int) (EntryFeeder, bool) {
	cuLang, ok := safeGetEntryVal[int64](cuEntry, dwarf.AttrLanguage, "compile unit language")
	if !ok {
		return nil, false
	}

	pkg := k.GetPackageFromDwarfCompileUnit(cuEntry)
	if pkg == nil {
		return nil, false
	}

	readFileName := dwarfutil.EntryFileReader(cuEntry, d)

	return func(e *dwarf.Entry) {
		switch e.Tag {
		case dwarf.TagSubprogram:
			k.AddDwarfSubProgram(cuLang == dwarfutil.DwLangGo, d, e, pkg, readFileName)
		case dwarf.TagVariable:
			k.AddDwarfVariable(e, d, pkg, ptrSize)
		}
	}, true
}

func (k *KnownInfo) TryLoadDwarf() bool {
	d, err := k.Wrapper.DWARF()
	if err != nil {
		slog.Debug(fmt.Sprintf("Failed to load DWARF: %v", err))
		return false
	}

	goarch := k.Wrapper.GoArch()
	var ptrSize int
	switch goarch {
	case "386", "arm":
		ptrSize = 4
	default:
		ptrSize = 8
	}

	k.HasDWARF = true

	r := d.Reader()

	type item struct {
		feeder EntryFeeder
		entry  *dwarf.Entry
	}

	entryChan := make(chan item, 256)
	processing := sync.WaitGroup{}
	processing.Add(1)
	go func() {
		defer processing.Done()
		for i := range entryChan {
			if !dwarfutil.EntryShouldIgnore(i.entry) {
				i.feeder(i.entry)
			}
		}
	}()

	var feeder EntryFeeder
	var entry *dwarf.Entry

	for entry, err = r.Next(); entry != nil; entry, err = r.Next() {
		if err != nil {
			slog.Warn(fmt.Sprintf("Failed to load DWARF: %v", err))
			return false
		}

		switch entry.Tag {
		case dwarf.TagCompileUnit:
			var ok bool
			feeder, ok = k.GetDwarfCompileUnitFeeder(d, entry, ptrSize)
			if !ok {
				r.SkipChildren()
			}
		case dwarf.TagSubprogram:
			entryChan <- item{
				feeder: feeder,
				entry:  entry,
			}
			r.SkipChildren()
		case dwarf.TagVariable:
			entryChan <- item{
				feeder: feeder,
				entry:  entry,
			}
		}
	}

	close(entryChan)
	processing.Wait()

	return true
}
