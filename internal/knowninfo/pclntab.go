package knowninfo

import (
	"fmt"
	"log/slog"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (k *KnownInfo) attributeGeneratedData(pkg *entity.Package, name string, addr, size uint64) uint64 {
	if size == 0 {
		return 0
	}

	sym := entity.NewSymbol(name, addr, size, entity.AddrTypeData)
	ap := k.KnownAddr.InsertSymbol(sym, pkg, entity.AddrSourceGoPclntab)
	if ap == nil {
		return 0
	}
	pkg.AddSymbol(sym, ap)
	return size
}

// AnalyzePclntabMeta analyzes pclntab sub-tables that are not covered by
// per-function PclnSymbolSize accounting. This includes the funcnametab,
// cutab, filetab overhead tables, and distributes the ftab region
// proportionally across packages.
func (k *KnownInfo) AnalyzePclntabMeta() error {
	slog.Info("Analyzing pclntab sub-table metadata...")

	md, err := k.Gore.Moduledata()
	if err != nil {
		return fmt.Errorf("pclntab meta analysis moduledata: %w", err)
	}

	if err := k.loadExactPclnRanges(); err != nil {
		return err
	}

	// Create virtual package for pclntab overhead sub-tables.
	pclntabPkg := k.getOrCreateVirtualPackage("runtime/pclntab", entity.PackageTypeGenerated)

	cutab := md.Cutab()
	filetab := md.Filetab()
	pctab := md.Pctab()

	var totalAttributed uint64

	// funcnametab is NOT attributed here — it is already covered by
	// per-function PclnSymbolSize.Name (FuncNameSize) accounting.

	// cutab: from cutab.Address to filetab.Address
	if cutab.Address > 0 && filetab.Address > 0 && filetab.Address > cutab.Address {
		size := filetab.Address - cutab.Address
		totalAttributed += k.attributeGeneratedData(pclntabPkg, "pclntab:cutab", cutab.Address, size)
	}

	// filetab: from filetab.Address to pctab.Address
	if filetab.Address > 0 && pctab.Address > 0 && pctab.Address > filetab.Address {
		size := pctab.Address - filetab.Address
		totalAttributed += k.attributeGeneratedData(pclntabPkg, "pclntab:filetab", filetab.Address, size)
	}

	// pctab and functab are skipped — already covered by per-function
	// PclnSymbolSize and the ftab attribution above.

	slog.Info("Pclntab sub-table metadata analyzed", "totalAttributed", totalAttributed)

	return nil
}
