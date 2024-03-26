package internal

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"log/slog"
	"strings"
)

func (k *KnownInfo) MarkSymbol(name string, addr, size uint64, typ entity.AddrType) error {
	var pkg *entity.Package
	pkgName := k.ExtractPackageFromSymbol(name)

	switch {
	case pkgName == "" || strings.HasPrefix(name, "x_cgo"):
		// we assume it's a cgo symbol
		pkgName = "cgo"
		pkg = nil
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

	k.KnownAddr.InsertSymbol(addr, size, pkg, typ, entity.SymbolMeta{
		SymbolName:  utils.Deduplicate(name),
		PackageName: utils.Deduplicate(pkgName),
	})

	return nil
}
