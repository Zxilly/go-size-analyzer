package internal

import (
	"log/slog"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

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
