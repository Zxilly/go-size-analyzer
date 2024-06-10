package knowninfo

import (
	"debug/dwarf"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ZxillyFork/gore"
	"github.com/ZxillyFork/gosym"
	"github.com/go-delve/delve/pkg/dwarf/op"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

func (k *KnownInfo) AddrForDWARFVar(entry *dwarf.Entry) (uint64, error) {
	arch := k.Wrapper.GoArch()
	var ptrSize int
	switch arch {
	case "386", "arm":
		ptrSize = 4
	default:
		ptrSize = 8
	}

	instsAny := entry.Val(dwarf.AttrLocation)
	if instsAny == nil {
		return 0, errors.New("no location attribute")
	}
	insts, ok := instsAny.([]byte)
	if !ok {
		return 0, errors.New("failed to cast location attribute to []byte")
	}

	imageBase := uint64(0)
	if peWrapper, ok := k.Wrapper.(*wrapper.PeWrapper); ok {
		imageBase = peWrapper.ImageBase
	}

	addr, _, err := op.ExecuteStackProgram(op.DwarfRegisters{StaticBase: imageBase}, insts, ptrSize, nil)
	if err != nil {
		return 0, err
	}

	return uint64(addr), nil
}

func SizeForDWARFVar(d *dwarf.Data, entry *dwarf.Entry) (uint64, error) {
	sizeOffset := entry.Val(dwarf.AttrType).(dwarf.Offset)

	typ, err := d.Type(sizeOffset)
	if err != nil {
		return 0, err
	}

	return uint64(typ.Size()), nil
}

func (k *KnownInfo) TryLoadDwarf() bool {
	d, err := k.Wrapper.DWARF()
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to load DWARF: %v", err))
		return false
	}

	k.HasDWARF = true

	r := d.Reader()

	langPackages := make(map[string]*entity.Package)

	loadCompilerUnit := func(cuEntry *dwarf.Entry, pendingEntry []*dwarf.Entry) {
		var pkg *entity.Package
		var filePath string

		cuLang, _ := cuEntry.Val(dwarf.AttrLanguage).(int64)
		cuName, _ := cuEntry.Val(dwarf.AttrName).(string)

		if cuLang == DwLangGo {
			// if we have load it with pclntab?
			pkg = k.Deps.Trie.Get(cuName)
			if pkg == nil {
				pkg = entity.NewPackage()
				pkg.Name = cuName
			}
			pkg.DwarfEntry = cuEntry
			filePath = "<unknown>"
			typ := entity.PackageTypeVendor
			if cuName == "main" {
				typ = entity.PackageTypeMain
			} else if gore.IsStandardLibrary(cuName) {
				typ = entity.PackageTypeStd
			}
			pkg.Type = typ
		} else {
			pkgName := fmt.Sprintf("CGO %s", LanguageString(cuLang))
			pkg = langPackages[pkgName]
			if pkg == nil {
				pkg = entity.NewPackage()
				pkg.Name = pkgName
				pkg.Type = entity.PackageTypeCGO
				langPackages[pkgName] = pkg
			}

			compDirAny := cuEntry.Val(dwarf.AttrCompDir)
			if compDirAny == nil {
				slog.Warn("Failed to load DWARF: no compDir")
				return
			}
			compDir := compDirAny.(string)

			filePath = fmt.Sprintf("%s/%s", compDir, cuName)
		}

		for _, subEntry := range pendingEntry {
			subEntryName := subEntry.Val(dwarf.AttrName).(string)

			if subEntry.Tag == dwarf.TagVariable {
				addr, err := k.AddrForDWARFVar(subEntry)
				if err != nil {
					slog.Warn(fmt.Sprintf("Failed to load DWARF var %s: %v", subEntryName, err))
					for _, field := range subEntry.Field {
						slog.Debug(fmt.Sprintf("%#v", field))
					}
					continue
				}
				size, err := SizeForDWARFVar(d, subEntry)
				if err != nil {
					slog.Warn(fmt.Sprintf("Failed to load DWARF var %s: %v", subEntryName, err))
					for _, field := range subEntry.Field {
						slog.Debug(fmt.Sprintf("%#v", field))
					}
					continue
				}

				ap := k.KnownAddr.InsertSymbol(addr, size, pkg, entity.AddrTypeData, entity.SymbolMeta{
					SymbolName:  utils.Deduplicate(subEntryName),
					PackageName: utils.Deduplicate(pkg.Name),
				})

				pkg.AddSymbol(addr, size, entity.AddrTypeData, subEntryName, ap)
			} else if subEntry.Tag == dwarf.TagSubprogram {
				ranges, err := d.Ranges(subEntry)
				if err != nil {
					slog.Warn(fmt.Sprintf("Failed to load DWARF function size: %v", err))
					continue
				}

				if len(ranges) == 0 {
					// fixme: maybe compiler optimization it?
					// example: sqlite3 simpleDestroy
					continue
				}

				addr := ranges[0][0]
				size := ranges[0][1] - ranges[0][0]

				typ := entity.FuncTypeFunction
				receiverName := ""
				if cuLang == DwLangGo {
					receiverName = (&gosym.Sym{Name: subEntryName}).ReceiverName()
					if receiverName != "" {
						typ = entity.FuncTypeMethod
					}
				}

				fn := &entity.Function{
					Name:     subEntryName,
					Addr:     addr,
					CodeSize: size,
					Type:     typ,
					Receiver: receiverName,
					PclnSize: entity.NewEmptyPclnSymbolSize(),
				}

				added := pkg.AddFuncIfNotExists(filePath, fn)

				if added {
					k.KnownAddr.Text.Insert(&entity.Addr{
						AddrPos:    &entity.AddrPos{Addr: addr, Size: size, Type: entity.AddrTypeText},
						Pkg:        pkg,
						Function:   fn,
						SourceType: entity.AddrSourceDwarf,
						Meta:       entity.DwarfMeta{},
					})
				}
			} else {
				panic("unreachable")
			}
		}
	}

	shouldIgnore := func(entry *dwarf.Entry) bool {
		declaration := entry.Val(dwarf.AttrDeclaration)
		if declaration != nil {
			val := declaration.(bool)
			if val {
				return true
			}
		}

		inline := entry.Val(dwarf.AttrInline)
		if inline != nil {
			val := inline.(int64)
			if val > 0 {
				return true
			}
		}

		abstractOrigin := entry.Val(dwarf.AttrAbstractOrigin)
		if abstractOrigin != nil {
			return true
		}

		specification := entry.Val(dwarf.AttrSpecification)

		return specification != nil
	}

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
				loadCompilerUnit(cuEntry, pendingEntry)
				cuEntry = nil
				pendingEntry = nil
			}
		}

		switch entry.Tag {
		case dwarf.TagCompileUnit:
			cuEntry = entry
		case dwarf.TagSubprogram:
			if !shouldIgnore(entry) {
				pendingEntry = append(pendingEntry, entry)
			}
		case dwarf.TagVariable:
			if !shouldIgnore(entry) && depth == 2 {
				pendingEntry = append(pendingEntry, entry)
			}
		}

		if entry.Children {
			depth++
		}
	}

	// add langPackages to knownPackages
	for _, pkg := range langPackages {
		k.Deps.Trie.Put(pkg.Name, pkg)
	}

	return true
}
