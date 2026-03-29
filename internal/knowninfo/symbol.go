package knowninfo

import (
	"log/slog"
	"strings"

	"github.com/ZxillyFork/gosym"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

// ExtractPackageFromSymbol copied from debug/gosym/symtab.go
func (k *KnownInfo) ExtractPackageFromSymbol(s string) string {
	var ver gosym.Version
	if k.VersionFlag.Meq120 {
		ver = gosym.Ver120 // ver120
	} else if k.VersionFlag.Leq118 {
		ver = gosym.Ver118 // ver118
	}

	sym := &gosym.Sym{
		Name:      s,
		GoVersion: ver,
	}

	packageName := sym.PackageName()

	return utils.UglyGuess(packageName)
}

func (k *KnownInfo) getOrCreateVirtualPackage(name string, typ entity.PackageType) *entity.Package {
	pkg, ok := k.Deps.GetPackage(name)
	if ok {
		return pkg
	}
	pkg = entity.NewPackage()
	pkg.Name = name
	pkg.Type = typ
	k.Deps.Trie.Put(name, pkg)
	return pkg
}

// resolvePackage looks up a package by exact match, then prefix match,
// then creates a virtual package as fallback.
func (k *KnownInfo) resolvePackage(pkgPath string, fallbackType entity.PackageType) *entity.Package {
	if pkg, ok := k.Deps.GetPackage(pkgPath); ok {
		return pkg
	}
	if pkg, ok := k.Deps.GetPackageByPrefixMatch(pkgPath); ok {
		return pkg
	}
	return k.getOrCreateVirtualPackage(pkgPath, fallbackType)
}

func (k *KnownInfo) MarkSymbol(name string, addr, size uint64, typ entity.AddrType) {
	if typ != entity.AddrTypeData {
		// todo: support text symbols, cross check with pclntab
		return
	}

	var pkg *entity.Package
	pkgName := k.ExtractPackageFromSymbol(name)

	switch {
	case strings.HasPrefix(name, "type:.namedata.") || strings.HasPrefix(name, "type:.importpath."):
		pkg = k.getOrCreateVirtualPackage("runtime/types", entity.PackageTypeGenerated)
	case pkgName == "" || strings.HasPrefix(name, "x_cgo"):
		pkg = k.getOrCreateVirtualPackage("CGO", entity.PackageTypeCGO)
	case pkgName == "$f64" || pkgName == "$f32":
		return
	default:
		var ok bool
		pkg, ok = k.Deps.GetPackage(pkgName)
		if !ok {
			// try longest prefix match as fallback
			pkg, ok = k.Deps.GetPackageByPrefixMatch(pkgName)
			if !ok {
				slog.Debug("package not found", "name", pkgName, "symbol", name, "type", typ)
				return
			}
		}
	}

	symbol := entity.NewSymbol(name, addr, size, typ)

	ap := k.KnownAddr.InsertSymbol(symbol, pkg)
	if ap == nil {
		return
	}
	pkg.AddSymbol(symbol, ap)
}

// CapturePclntabSymbolAddrs looks up known pclntab sub-table symbols by name
// and stores their addresses for later use by pclntab metadata analysis.
func (k *KnownInfo) CapturePclntabSymbolAddrs() {
	lookup := func(name string) uint64 {
		sym, err := k.Gore.GetSymbol(name)
		if err != nil {
			return 0
		}
		return sym.Value
	}

	k.PclntabSyms = PclntabMeta{
		FuncnametabAddr: lookup("runtime.funcnametab"),
		CutabAddr:       lookup("runtime.cutab"),
		FiletabAddr:     lookup("runtime.filetab"),
		PctabAddr:       lookup("runtime.pctab"),
		FunctabAddr:     lookup("runtime.functab"),
		PclntabEnd:      lookup("runtime.epclntab"),
	}
}

func (k *KnownInfo) AnalyzeSymbol(store bool) error {
	slog.Info("Analyzing symbols...")

	marker := k.MarkSymbol
	if !store {
		marker = nil
	}

	err := k.Wrapper.LoadSymbols(marker, func(addr, size uint64) {
		k.GoStringSymbol = &entity.AddrPos{
			Addr: addr,
			Size: size,
			Type: entity.AddrTypeData,
		}
	})
	if err != nil {
		return err
	}

	slog.Info("Analyzing symbols done")

	return nil
}
