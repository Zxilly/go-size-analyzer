package knowninfo

import (
	"debug/dwarf"
	"fmt"
	"log/slog"
	"math"

	"github.com/ZxillyFork/gore"
	"github.com/ZxillyFork/gosym"
	"github.com/go-delve/delve/pkg/dwarf/op"

	dwarfutil "github.com/Zxilly/go-size-analyzer/internal/dwarf"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func (k *KnownInfo) AddDwarfVariable(entry *dwarf.Entry, d *dwarf.Data, pkg *entity.Package, ptrSize int) {
	instsAny := entry.Val(dwarf.AttrLocation)
	if instsAny == nil {
		slog.Warn(fmt.Sprintf("no location attribute for %s", dwarfutil.EntryPrettyPrinter(entry)))
		return
	}
	insts, ok := instsAny.([]byte)
	if !ok {
		slog.Warn(fmt.Sprintf("location attribute is not []byte for %s", dwarfutil.EntryPrettyPrinter(entry)))
		return
	}

	addr, _, err := op.ExecuteStackProgram(op.DwarfRegisters{StaticBase: k.Wrapper.ImageBase()}, insts, ptrSize, nil)
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to execute location attribute for %s: %v", dwarfutil.EntryPrettyPrinter(entry), err))
		return
	}

	contents, typSize, err := dwarfutil.SizeForDWARFVar(d, entry, func(addrCb, size uint64) ([]byte, error) {
		if addrCb == math.MaxUint64 {
			addrCb = uint64(addr)
		}

		return k.Wrapper.ReadAddr(addrCb, size)
	})
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to load DWARF var %s: %v", dwarfutil.EntryPrettyPrinter(entry), err))
		return
	}

	entryName := utils.Deduplicate(entry.Val(dwarf.AttrName).(string))

	ap := k.KnownAddr.InsertSymbolFromDWARF(uint64(addr), typSize, pkg, entity.AddrTypeData, entity.SymbolMeta{
		SymbolName:  entryName,
		PackageName: utils.Deduplicate(pkg.Name),
	})

	pkg.AddSymbol(uint64(addr), typSize, entity.AddrTypeData, entryName, ap)

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
			ap = k.KnownAddr.InsertSymbolFromDWARF(content.Addr, content.Size, pkg, entity.AddrTypeData, entity.SymbolMeta{
				SymbolName:  valueName,
				PackageName: utils.Deduplicate(pkg.Name),
			})

			pkg.AddSymbol(content.Addr, content.Size, entity.AddrTypeData, valueName, ap)
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
	subEntryName := subEntry.Val(dwarf.AttrName).(string)

	ranges, err := d.Ranges(subEntry)
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to load DWARF function size: %v", err))
		return
	}

	if len(ranges) == 0 {
		// fixme: maybe compiler optimize it?
		// example: sqlite3 simpleDestroy
		slog.Warn(fmt.Sprintf("Failed to load DWARF function size, no range: %s", subEntryName))
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

	added := pkg.AddFuncIfNotExists(filename, fn)

	if added {
		k.KnownAddr.Text.Insert(&entity.Addr{
			AddrPos:    &entity.AddrPos{Addr: addr, Size: size, Type: entity.AddrTypeText},
			Pkg:        pkg,
			Function:   fn,
			SourceType: entity.AddrSourceDwarf,
			Meta:       entity.DwarfMeta{},
		})
	}
}

func (k *KnownInfo) GetPackageFromDwarfCompileUnit(cuEntry *dwarf.Entry) *entity.Package {
	cuLang, _ := cuEntry.Val(dwarf.AttrLanguage).(int64)
	cuName, _ := cuEntry.Val(dwarf.AttrName).(string)

	var pkg *entity.Package

	if cuLang == dwarfutil.DwLangGo {
		// if we have load it with pclntab?
		pkg = k.Deps.Trie.Get(cuName)
		if pkg == nil {
			pkg = entity.NewPackage()
			pkg.Name = cuName
		}
		pkg.SetDwarfEntry(cuEntry)
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

func (k *KnownInfo) LoadDwarfCompileUnit(d *dwarf.Data, cuEntry *dwarf.Entry, pendingEntry []*dwarf.Entry, ptrSize int) {
	cuLang, _ := cuEntry.Val(dwarf.AttrLanguage).(int64)

	pkg := k.GetPackageFromDwarfCompileUnit(cuEntry)

	readFileName := dwarfutil.EntryFileReader(cuEntry, d)

	for _, subEntry := range pendingEntry {
		switch subEntry.Tag {
		case dwarf.TagSubprogram:
			k.AddDwarfSubProgram(cuLang == dwarfutil.DwLangGo, d, subEntry, pkg, readFileName)
		case dwarf.TagVariable:
			k.AddDwarfVariable(subEntry, d, pkg, ptrSize)
		}
	}
}

func (k *KnownInfo) TryLoadDwarf() bool {
	d, err := k.Wrapper.DWARF()
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to load DWARF: %v", err))
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

	var cuEntry *dwarf.Entry
	var pendingEntry []*dwarf.Entry
	depth := 1

	for entry, err := r.Next(); entry != nil; entry, err = r.Next() {
		if err != nil {
			slog.Warn(fmt.Sprintf("Failed to load DWARF: %v", err))
			return false
		}

		if entry.Tag == 0 {
			depth--
			if depth <= 0 {
				panic("broken DWARF")
			}
			if depth == 1 && cuEntry != nil {
				k.LoadDwarfCompileUnit(d, cuEntry, pendingEntry, ptrSize)

				cuEntry = nil
				pendingEntry = nil
			}
		}

		switch entry.Tag {
		case dwarf.TagCompileUnit:
			cuEntry = entry
		case dwarf.TagSubprogram:
			if !dwarfutil.EntryShouldIgnore(entry) {
				pendingEntry = append(pendingEntry, entry)
			}
		case dwarf.TagVariable:
			if !dwarfutil.EntryShouldIgnore(entry) && depth == 2 {
				pendingEntry = append(pendingEntry, entry)
			}
		}

		if entry.Children {
			depth++
		}
	}

	return true
}
