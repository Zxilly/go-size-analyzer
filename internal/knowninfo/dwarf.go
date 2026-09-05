package knowninfo

import (
	"context"
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ZxillyFork/gore"
	"github.com/ZxillyFork/gosym"
	"github.com/go-delve/delve/pkg/dwarf/op"

	dwarfutil "github.com/Zxilly/go-size-analyzer/internal/dwarf"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func safeGetEntryVal[T any](entry *dwarf.Entry, attr dwarf.Attr, name string, quiet bool) (T, bool) {
	v, ok := entry.Val(attr).(T)
	if !ok {
		if !quiet {
			slog.Debug(fmt.Sprintf("Failed to load DWARF %s: %s", name, dwarfutil.EntryPrettyPrint(entry)))
		}
		return *new(T), false
	}
	return v, true
}

// Go emits a single DW_OP_addr for global variables. Delve's general
// expression interpreter decodes that operand as little-endian, regardless
// of DwarfRegisters.ByteOrder, so read this common case directly.
func staticVariableAddress(insts []byte, ptrSize int, order binary.ByteOrder) (uint64, error) {
	if len(insts) == 1+ptrSize && insts[0] == byte(op.DW_OP_addr) {
		switch ptrSize {
		case 4:
			return uint64(order.Uint32(insts[1:])), nil
		case 8:
			return order.Uint64(insts[1:]), nil
		}
	}
	if order != binary.LittleEndian {
		return 0, errors.New("unsupported big-endian DWARF location expression")
	}
	addr, _, err := op.ExecuteStackProgram(op.DwarfRegisters{ByteOrder: order}, insts, ptrSize, nil)
	return uint64(addr), err
}

func (k *KnownInfo) AddDwarfVariable(entry *dwarf.Entry, d *dwarf.Data, pkg *entity.Package, ptrSize int, isGo bool) {
	insts, ok := safeGetEntryVal[[]byte](entry, dwarf.AttrLocation, "location attribute", !isGo)
	if !ok {
		return
	}

	_, order := ptrSizeAndOrder(k.Wrapper.GoArch())
	addr, err := staticVariableAddress(insts, ptrSize, order)
	if err != nil {
		if !isGo {
			return
		}
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

	contents, typSize, err := dwarfutil.SizeForDWARFVar(d, entry, addr, dwarfutil.MemoryReader{
		Read: k.Wrapper.ReadAddr, ByteOrder: order,
	})
	if err != nil {
		if isGo {
			slog.Warn(fmt.Sprintf("Failed to load DWARF var %s: %v", dwarfutil.EntryPrettyPrint(entry), err))
		}
		return
	}

	entryName, ok := safeGetEntryVal[string](entry, dwarf.AttrName, "variable name", !isGo)
	if !ok {
		if isGo {
			slog.Debug(fmt.Sprintf("Failed to load DWARF var name: %s", dwarfutil.EntryPrettyPrint(entry)))
		}
		return
	}

	symbol := entity.NewSymbol(entryName, addr, typSize, entity.AddrTypeData)

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

			symbol = entity.NewSymbol(valueName, k.convertAddr(content.Addr), content.Size, entity.AddrTypeData)

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
	subEntryName, ok := safeGetEntryVal[string](subEntry, dwarf.AttrName, "function name", !isGo)
	if !ok {
		return
	}

	ranges, err := d.Ranges(subEntry)
	if err != nil {
		if isGo {
			slog.Debug(fmt.Sprintf("Failed to load DWARF function size: %v", err))
		}
		return
	}

	if len(ranges) == 0 {
		// fixme: maybe compiler optimize it?
		// example: sqlite3 simpleDestroy
		if isGo {
			slog.Debug(fmt.Sprintf("Failed to load DWARF function size, no range: %s", subEntryName))
		}
		return
	}

	// Functions may be split across non-contiguous ranges (PGO, inlining).
	addr := ranges[0][0]
	var size uint64
	for _, r := range ranges {
		size += r[1] - r[0]
	}

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
		for _, r := range ranges {
			k.KnownAddr.InsertTextFromDWARF(r[0], r[1]-r[0], fn)
		}
	}
}

func (k *KnownInfo) GetPackageFromDwarfCompileUnit(cuEntry *dwarf.Entry) *entity.Package {
	cuLang, ok := safeGetEntryVal[int64](cuEntry, dwarf.AttrLanguage, "compile unit language", false)
	if !ok {
		return nil
	}

	cuName, ok := safeGetEntryVal[string](cuEntry, dwarf.AttrName, "compile unit name", false)
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
		if gore.IsStandardLibrary(cuName) && cuName != "main" {
			typ = entity.PackageTypeStd
		} else if cuName == "main" || k.isMainModulePackage(cuName) {
			typ = entity.PackageTypeMain
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
	cuLang, ok := safeGetEntryVal[int64](cuEntry, dwarf.AttrLanguage, "compile unit language", false)
	if !ok {
		return nil, false
	}

	pkg := k.GetPackageFromDwarfCompileUnit(cuEntry)
	if pkg == nil {
		return nil, false
	}

	readFileName := dwarfutil.EntryFileReader(cuEntry, d)

	isGo := cuLang == dwarfutil.DwLangGo

	return func(e *dwarf.Entry) {
		switch e.Tag {
		case dwarf.TagSubprogram:
			k.AddDwarfSubProgram(isGo, d, e, pkg, readFileName)
		case dwarf.TagVariable:
			k.AddDwarfVariable(e, d, pkg, ptrSize, isGo)
		default:
		}
	}, true
}

func (k *KnownInfo) TryLoadDwarf() bool {
	d, err := k.Wrapper.DWARF()
	if err != nil {
		slog.Debug(fmt.Sprintf("Failed to load DWARF: %v", err))
		return false
	}

	ptrSize, _ := ptrSizeAndOrder(k.Wrapper.GoArch())

	// debug/dwarf.Data lazily caches unit and type information. Resolving
	// types concurrently with the main reader races on those shared caches.
	r := d.Reader()
	var feeder EntryFeeder
	for {
		entry, err := r.Next()
		if err != nil {
			slog.Warn("Failed to load DWARF", "error", err)
			return false
		}
		if entry == nil {
			break
		}
		switch entry.Tag {
		case dwarf.TagCompileUnit:
			var ok bool
			feeder, ok = k.GetDwarfCompileUnitFeeder(d, entry, ptrSize)
			if !ok {
				r.SkipChildren()
			}
		case dwarf.TagSubprogram, dwarf.TagVariable:
			if feeder != nil && !dwarfutil.EntryShouldIgnore(entry) {
				feeder(entry)
			}
			if entry.Tag == dwarf.TagSubprogram {
				r.SkipChildren()
			}
		}
	}
	k.HasDWARF = true
	return true
}
