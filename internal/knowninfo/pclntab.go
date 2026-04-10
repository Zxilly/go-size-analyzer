package knowninfo

import (
	"fmt"
	"log/slog"

	"github.com/ZxillyFork/gore"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

// attributeRangeByFunctionCount distributes the given pclntab range
// proportionally by function count per package.
func (k *KnownInfo) attributeRangeByFunctionCount(prefix string, addr, length uint64) uint64 {
	if length == 0 {
		return 0
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
		slog.Warn("No functions found for pclntab attribution", "prefix", prefix)
		return 0
	}

	var offset uint64
	var attributed uint64
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

		symName := fmt.Sprintf("%s[%s]", prefix, pc.name)
		sym := entity.NewSymbol(symName, addr+offset, size, entity.AddrTypeData)

		ap := k.KnownAddr.InsertSymbol(sym, pc.pkg)
		if ap != nil {
			pc.pkg.AddSymbol(sym, ap)
			attributed += size
		}

		offset += size
	}

	return attributed
}

func (k *KnownInfo) attributeGeneratedData(pkg *entity.Package, name string, addr, size uint64) uint64 {
	if size == 0 {
		return 0
	}

	sym := entity.NewSymbol(name, addr, size, entity.AddrTypeData)
	ap := k.KnownAddr.InsertSymbol(sym, pkg)
	if ap == nil {
		return 0
	}
	pkg.AddSymbol(sym, ap)
	return size
}

// attributeFtab distributes the ftab region proportionally by function count
// per package. Each package gets a synthetic data symbol sized proportionally
// to its share of all functions.
func (k *KnownInfo) attributeFtab(addr, length uint64) {
	attributed := k.attributeRangeByFunctionCount("pclntab:ftab", addr, length)
	slog.Info("Attributed ftab region", "totalSize", length, "attributed", attributed)
}

func (k *KnownInfo) analyzeWasmPclntabMeta(md gore.Moduledata) error {
	pclntab := md.PCLNTab()
	if pclntab.Length == 0 {
		slog.Warn("PCLNTab region not available")
		return nil
	}

	pclntabEnd := pclntab.Address + pclntab.Length
	if pclntabEnd < pclntab.Address {
		return fmt.Errorf("wasm pclntab range overflow: start=%#x len=%#x", pclntab.Address, pclntab.Length)
	}

	ftab := md.FuncTab()
	if ftab.Length == 0 {
		attributed := k.attributeRangeByFunctionCount("pclntab:functions", pclntab.Address, pclntab.Length)
		slog.Info("Wasm pclntab metadata analyzed", "totalAttributed", attributed)
		return nil
	}

	if ftab.Address < pclntab.Address || ftab.Address > pclntabEnd {
		return fmt.Errorf("wasm functab outside pclntab: functab=%#x pclntab=[%#x,%#x)", ftab.Address, pclntab.Address, pclntabEnd)
	}

	totalAttributed := uint64(0)
	pclntabPkg := k.getOrCreateVirtualPackage("runtime/pclntab", entity.PackageTypeGenerated)

	totalAttributed += k.attributeGeneratedData(pclntabPkg, "pclntab:meta", pclntab.Address, ftab.Address-pclntab.Address)
	totalAttributed += k.attributeRangeByFunctionCount("pclntab:functions", ftab.Address, pclntabEnd-ftab.Address)

	slog.Info("Wasm pclntab metadata analyzed", "totalAttributed", totalAttributed)
	return nil
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

	if k.Wrapper.GoArch() == "wasm" {
		return k.analyzeWasmPclntabMeta(md)
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
