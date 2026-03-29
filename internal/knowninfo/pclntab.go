package knowninfo

import (
	"fmt"
	"log/slog"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

// attributeFtab distributes the ftab region proportionally by function count
// per package. Each package gets a synthetic data symbol sized proportionally
// to its share of all functions.
func (k *KnownInfo) attributeFtab(addr, length uint64) {
	if length == 0 {
		return
	}

	// Count functions per package.
	type pkgCount struct {
		name string
		pkg  *entity.Package
		n    uint64
	}

	var (
		packages []pkgCount
		total    uint64
	)

	_ = k.Deps.Trie.Walk(func(name string, p *entity.Package) error {
		n := uint64(p.FuncCount())
		if n > 0 {
			packages = append(packages, pkgCount{name: name, pkg: p, n: n})
			total += n
		}
		return nil
	})

	if total == 0 {
		slog.Warn("No functions found for ftab attribution")
		return
	}

	var offset uint64
	for i, pc := range packages {
		var size uint64
		if i == len(packages)-1 {
			// Last package gets remainder to avoid rounding errors.
			size = length - offset
		} else {
			size = (pc.n * length) / total
		}
		if size == 0 {
			continue
		}

		symName := fmt.Sprintf("pclntab:ftab[%s]", pc.name)
		sym := entity.NewSymbol(symName, addr+offset, size, entity.AddrTypeData)

		ap := k.KnownAddr.InsertSymbol(sym, pc.pkg)
		if ap != nil {
			pc.pkg.AddSymbol(sym, ap)
		}

		offset += size
	}

	slog.Info("Attributed ftab region", "totalSize", length, "packages", len(packages))
}

// AnalyzePclntabMeta analyzes pclntab sub-tables that are not covered by
// per-function PclnSymbolSize accounting. This includes the funcnametab,
// cutab, filetab overhead tables, and distributes the ftab region
// proportionally across packages.
// Failures are non-fatal: errors are logged and nil is returned.
func (k *KnownInfo) AnalyzePclntabMeta() error {
	slog.Info("Analyzing pclntab sub-table metadata...")

	md, err := k.Gore.Moduledata()
	if err != nil {
		slog.Warn("Failed to get moduledata for pclntab analysis", "err", err)
		return nil
	}

	// Attribute ftab region proportionally.
	ftab := md.FuncTab()
	if ftab.Address > 0 && ftab.Length > 0 {
		k.attributeFtab(ftab.Address, ftab.Length)
	} else {
		slog.Warn("FuncTab region not available")
	}

	// Create virtual package for pclntab overhead sub-tables.
	pclntabPkg := k.getOrCreateVirtualPackage("runtime/pclntab", entity.PackageTypeGenerated)

	meta := k.PclntabSyms
	var totalAttributed uint64

	// funcnametab is NOT attributed here — it is already covered by
	// per-function PclnSymbolSize.Name (FuncNameSize) accounting.

	// cutab: from CutabAddr to FiletabAddr
	if meta.CutabAddr > 0 && meta.FiletabAddr > 0 && meta.FiletabAddr > meta.CutabAddr {
		size := meta.FiletabAddr - meta.CutabAddr
		sym := entity.NewSymbol("pclntab:cutab", meta.CutabAddr, size, entity.AddrTypeData)
		ap := k.KnownAddr.InsertSymbol(sym, pclntabPkg)
		if ap != nil {
			pclntabPkg.AddSymbol(sym, ap)
			totalAttributed += size
		}
	}

	// filetab: from FiletabAddr to PctabAddr
	if meta.FiletabAddr > 0 && meta.PctabAddr > 0 && meta.PctabAddr > meta.FiletabAddr {
		size := meta.PctabAddr - meta.FiletabAddr
		sym := entity.NewSymbol("pclntab:filetab", meta.FiletabAddr, size, entity.AddrTypeData)
		ap := k.KnownAddr.InsertSymbol(sym, pclntabPkg)
		if ap != nil {
			pclntabPkg.AddSymbol(sym, ap)
			totalAttributed += size
		}
	}

	// pctab and functab are skipped — already covered by per-function
	// PclnSymbolSize and the ftab attribution above.

	slog.Info("Pclntab sub-table metadata analyzed", "totalAttributed", totalAttributed)

	return nil
}
