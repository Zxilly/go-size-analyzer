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

func (k *KnownInfo) MarkSymbol(name string, addr, size uint64, typ entity.AddrType) {
	if typ != entity.AddrTypeData {
		// todo: support text symbols, cross check with pclntab
		// and further work on cgo symbols
		return
	}

	var pkg *entity.Package
	pkgName := k.ExtractPackageFromSymbol(name)

	switch {
	case pkgName == "" || strings.HasPrefix(name, "x_cgo"):
		// we assume it's a cgo symbol
		return // todo: implement cgo analysis in the future
	case pkgName == "$f64" || pkgName == "$f32":
		return
	default:
		var ok bool
		pkg, ok = k.Deps.GetPackage(pkgName)
		if !ok {
			slog.Debug("package not found", "name", pkgName, "symbol", name, "type", typ)
			return // no package found, skip
		}
	}

	symbol := entity.NewSymbol(name, addr, size, typ)

	ap := k.KnownAddr.InsertSymbol(symbol, pkg)
	if ap == nil {
		return
	}
	pkg.AddSymbol(symbol, ap)
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
