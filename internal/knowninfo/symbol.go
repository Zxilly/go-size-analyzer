package knowninfo

import (
	"github.com/ZxillyFork/gosym"
	"log/slog"
	"strings"

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

func (k *KnownInfo) MarkSymbol(name string, addr, size uint64, typ entity.AddrType) error {
	if typ != entity.AddrTypeData {
		// todo: support text symbols, cross check with pclntab
		// and further work on cgo symbols
		return nil
	}

	var pkg *entity.Package
	pkgName := k.ExtractPackageFromSymbol(name)

	switch {
	case pkgName == "" || strings.HasPrefix(name, "x_cgo"):
		// we assume it's a cgo symbol
		return nil // todo: implement cgo analysis in the future
	case pkgName == "$f64" || pkgName == "$f32":
		return nil
	default:
		var ok bool
		pkg, ok = k.Deps.GetPackage(pkgName)
		if !ok {
			slog.Debug("package not found", "name", pkgName, "symbol", name, "type", typ)
			return nil // no package found, skip
		}
	}

	ap := k.KnownAddr.InsertSymbol(addr, size, pkg, typ, entity.SymbolMeta{
		SymbolName:  utils.Deduplicate(name),
		PackageName: utils.Deduplicate(pkgName),
	})

	pkg.AddSymbol(addr, size, typ, name, ap)

	return nil
}

func (k *KnownInfo) AnalyzeSymbol() error {
	slog.Info("Analyzing symbols...")

	err := k.Wrapper.LoadSymbols(k.MarkSymbol)
	if err != nil {
		return err
	}

	k.KnownAddr.BuildSymbolCoverage()

	slog.Info("Analyzing symbols done")

	return nil
}
