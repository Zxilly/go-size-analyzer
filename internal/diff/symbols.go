package diff

import (
	"cmp"
	"fmt"
	"slices"
)

// Symbol sizes describe reported ranges, which may be shared references or
// generated metadata. They are not additional file bytes or deletion savings.
type diffSymbol struct {
	Package    string      `json:"package"`
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	ChangeType changeType  `json:"change_type"`
	Size       *byteChange `json:"size,omitempty"`
}

type symbolKey struct{ name, typ string }

type symbolInventory struct {
	symbols  []commonSymbol
	complete bool
}

func symbolInventories(r *commonResult) (map[string]symbolInventory, bool) {
	out := map[string]symbolInventory{}
	complete := r.Packages != nil
	var visit func(map[string]commonPackage)
	visit = func(packages map[string]commonPackage) {
		for key, pkg := range packages {
			name := pkg.Name
			if name == "" {
				name = key
			}
			item := symbolInventory{complete: pkg.Symbols != nil && pkg.SubPackages != nil}
			if pkg.Symbols != nil {
				item.symbols = append(item.symbols, (*pkg.Symbols)...)
				for _, symbol := range item.symbols {
					if symbol.Name == "" || symbol.Type == "" {
						item.complete = false
					}
				}
			}
			if previous, exists := out[name]; exists {
				item.symbols = append(item.symbols, previous.symbols...)
				item.complete = false
			}
			out[name] = item
			complete = complete && item.complete
			visit(pkg.SubPackages)
		}
	}
	visit(r.Packages)
	return out, complete
}

// DWARF and the native symbol table can expose the same declaration twice.
// Deduplicate only identical, known ranges inside one report. Addresses never
// participate in matching between builds; different ranges remain ambiguous.
func indexSymbols(symbols []commonSymbol) map[symbolKey][]commonSymbol {
	index := map[symbolKey][]commonSymbol{}
	type knownRange struct {
		symbolKey
		addr, size uint64
	}
	seen := map[knownRange]bool{}
	for _, s := range symbols {
		if s.Name == "" || s.Type == "" {
			continue
		}
		key := symbolKey{name: s.Name, typ: s.Type}
		if s.Addr != nil && s.Size != nil {
			r := knownRange{symbolKey: key, addr: *s.Addr, size: *s.Size}
			if seen[r] {
				continue
			}
			seen[r] = true
		}
		index[key] = append(index[key], s)
	}
	return index
}

func processSymbols(next, old *commonResult) ([]diffSymbol, []diagnostic) {
	nextPackages, nextComplete := symbolInventories(next)
	oldPackages, oldComplete := symbolInventories(old)
	var changes []diffSymbol
	var notes []diagnostic
	if !nextComplete || !oldComplete {
		notes = append(notes, diagnostic{Code: "symbol_details_unavailable", Message: "Symbol inventories are incomplete in at least one report; missing lists do not imply symbol deletion."})
	}
	packages := map[string]bool{}
	for key := range nextPackages {
		packages[key] = true
	}
	for key := range oldPackages {
		packages[key] = true
	}
	missingSize := false
	for pkg := range packages {
		n, hasNew := nextPackages[pkg]
		o, hasOld := oldPackages[pkg]
		if !hasNew {
			n.complete = nextComplete
		}
		if !hasOld {
			o.complete = oldComplete
		}
		newIndex, oldIndex := indexSymbols(n.symbols), indexSymbols(o.symbols)
		keys := map[symbolKey]bool{}
		for key := range newIndex {
			keys[key] = true
		}
		for key := range oldIndex {
			keys[key] = true
		}
		for key := range keys {
			a, b := oldIndex[key], newIndex[key]
			if len(a) > 1 || len(b) > 1 {
				notes = append(notes, diagnostic{
					Code: "symbol_identity_collision", Package: pkg, Identity: key.name,
					Message: fmt.Sprintf("Symbol identity (%s) has %d old and %d new distinct or unknown ranges; ambiguous entries were not compared.", key.typ, len(a), len(b)),
				})
				continue
			}
			if (len(a) == 0 || len(b) == 0) && (!o.complete || !n.complete) {
				continue
			}
			var before, after *uint64
			status := changeTypeChange
			if len(a) == 1 {
				before = a[0].Size
			} else {
				status = changeTypeAdd
			}
			if len(b) == 1 {
				after = b[0].Size
			} else {
				status = changeTypeRemove
			}
			change := metricChange(before, after, len(a) == 0, len(b) == 0)
			if change == nil {
				missingSize = true
			}
			if status != changeTypeChange || change != nil && change.Delta != 0 {
				changes = append(changes, diffSymbol{Package: pkg, Name: key.name, Type: key.typ, ChangeType: status, Size: change})
			}
		}
	}
	if missingSize {
		notes = append(notes, diagnostic{Code: "symbol_metric_unavailable", Message: "Some symbol sizes are missing or exceed the signed delta range; unknown values are omitted, not treated as zero."})
	}
	if len(changes) > 0 {
		notes = append(notes, diagnostic{Code: "symbol_metrics_overlap", Message: "Symbol sizes are reported ranges, including shared references and generated metadata. They can overlap each other and functions; do not sum them as file growth or deletion savings."})
	}
	slices.SortFunc(changes, func(a, b diffSymbol) int {
		if n := cmp.Compare(symbolImpact(b), symbolImpact(a)); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Package, b.Package); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		return cmp.Compare(a.Type, b.Type)
	})
	return changes, notes
}

func symbolImpact(s diffSymbol) uint64 {
	if s.Size == nil {
		return 0
	}
	if s.Size.Delta < 0 {
		return uint64(-s.Size.Delta)
	}
	return uint64(s.Size.Delta)
}
