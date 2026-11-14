package diff

import (
	"cmp"
	"fmt"
	"math"
	"slices"
)

type diagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Package  string `json:"package,omitempty"`
	Identity string `json:"identity,omitempty"`
}

type byteChange struct {
	From  uint64 `json:"from"`
	To    uint64 `json:"to"`
	Delta int64  `json:"delta"`
}

type diffFunction struct {
	Package         string      `json:"package"`
	Name            string      `json:"name"`
	FullName        string      `json:"full_name,omitempty"`
	Type            string      `json:"type"`
	Receiver        string      `json:"receiver,omitempty"`
	ChangeType      changeType  `json:"change_type"`
	CodeSize        *byteChange `json:"code_size,omitempty"`
	PclnBytes       *byteChange `json:"pcln_bytes,omitempty"`
	ReferencedBytes *byteChange `json:"referenced_bytes,omitempty"`
}

type functionInventory struct {
	functions []commonFunction
	complete  bool
}

type functionKey struct {
	full     bool
	typ      string
	name     string
	receiver string
}

func functionType(f commonFunction) string {
	if f.Type != "" {
		return f.Type
	}
	if f.Receiver != "" {
		return "method"
	}
	return "function"
}

func primaryKey(f commonFunction) functionKey {
	if f.FullName != "" {
		return functionKey{full: true, typ: functionType(f), name: f.FullName}
	}
	return legacyKey(f)
}

func legacyKey(f commonFunction) functionKey {
	return functionKey{typ: functionType(f), name: f.Name, receiver: f.Receiver}
}

func displayFunction(f commonFunction) string {
	if f.FullName != "" {
		return f.FullName
	}
	if f.Receiver != "" {
		return f.Receiver + "." + f.Name
	}
	return f.Name
}

// Missing files/functions in compact and legacy JSON are unknown, not empty.
// An absent package implies absent functions only if the other report includes
// a complete function inventory throughout its package tree.
func inventories(r *commonResult) (map[string]functionInventory, bool) {
	out := map[string]functionInventory{}
	complete := r.Packages != nil
	var visit func(map[string]commonPackage)
	visit = func(packages map[string]commonPackage) {
		for key, p := range packages {
			name := p.Name
			if name == "" {
				name = key
			}
			item := functionInventory{complete: p.Files != nil && p.SubPackages != nil}
			if p.Files != nil {
				for _, f := range *p.Files {
					if f.Functions == nil {
						item.complete = false
						continue
					}
					item.functions = append(item.functions, (*f.Functions)...)
					for _, fn := range *f.Functions {
						if fn.Name == "" && fn.FullName == "" {
							item.complete = false
						}
					}
				}
			}
			if previous, exists := out[name]; exists {
				item.functions = append(item.functions, previous.functions...)
				item.complete = false
			}
			out[name] = item
			complete = complete && item.complete
			visit(p.SubPackages)
		}
	}
	visit(r.Packages)
	return out, complete
}

func metricChange(old, next *uint64, oldAbsent, newAbsent bool) *byteChange {
	zero := uint64(0)
	if oldAbsent {
		old = &zero
	}
	if newAbsent {
		next = &zero
	}
	if old == nil || next == nil || *old > math.MaxInt64 || *next > math.MaxInt64 {
		return nil
	}
	return &byteChange{From: *old, To: *next, Delta: int64(*next) - int64(*old)}
}

func makeFunctionChange(pkg string, old, next *commonFunction) diffFunction {
	oldAbsent, newAbsent := old == nil, next == nil
	var empty commonFunction
	if oldAbsent {
		old = &empty
	}
	if newAbsent {
		next = &empty
	}
	f := next
	status := changeTypeChange
	if oldAbsent {
		status = changeTypeAdd
	}
	if newAbsent {
		status = changeTypeRemove
		f = old
	}
	return diffFunction{
		Package: pkg, Name: f.Name, FullName: f.FullName, Type: functionType(*f), Receiver: f.Receiver, ChangeType: status,
		CodeSize:        metricChange(old.CodeSize, next.CodeSize, oldAbsent, newAbsent),
		PclnBytes:       metricChange(old.PclnBytes, next.PclnBytes, oldAbsent, newAbsent),
		ReferencedBytes: metricChange(old.ReferencedBytes, next.ReferencedBytes, oldAbsent, newAbsent),
	}
}

func changedFunction(f diffFunction) bool {
	if f.ChangeType != changeTypeChange {
		return true
	}
	for _, m := range []*byteChange{f.CodeSize, f.PclnBytes, f.ReferencedBytes} {
		if m != nil && m.Delta != 0 {
			return true
		}
	}
	return false
}

func processFunctions(next, old *commonResult) ([]diffFunction, []diagnostic) {
	newPackages, newComplete := inventories(next)
	oldPackages, oldComplete := inventories(old)
	var changes []diffFunction
	var notes []diagnostic
	if !newComplete || !oldComplete {
		notes = append(notes, diagnostic{Code: "function_details_unavailable", Message: "Function details are incomplete in at least one report; additions and removals are only inferred where both inventories are complete."})
	}
	keys := map[string]bool{}
	for k := range newPackages {
		keys[k] = true
	}
	for k := range oldPackages {
		keys[k] = true
	}
	packageNames := make([]string, 0, len(keys))
	for k := range keys {
		packageNames = append(packageNames, k)
	}
	slices.Sort(packageNames)
	missingMetrics := map[string]bool{}
	for _, pkg := range packageNames {
		n, hasNew := newPackages[pkg]
		o, hasOld := oldPackages[pkg]
		if !hasNew {
			n.complete = newComplete
		}
		if !hasOld {
			o.complete = oldComplete
		}
		compared, warnings := comparePackageFunctions(pkg, o, n, missingMetrics)
		changes = append(changes, compared...)
		notes = append(notes, warnings...)
	}
	for _, metric := range []string{"code_size", "pcln_bytes", "referenced_bytes"} {
		if missingMetrics[metric] {
			notes = append(notes, diagnostic{Code: "function_metric_unavailable", Message: "Some functions lack comparable " + metric + " measurements or exceed the signed delta range; unavailable values are omitted, not treated as zero."})
		}
	}
	if len(changes) > 0 {
		notes = append(notes, diagnostic{Code: "function_metrics_overlap", Message: "Function code, pcln and referenced-data measurements can overlap between functions; do not sum them as file growth or deletion savings."})
	}
	slices.SortFunc(changes, func(a, b diffFunction) int {
		if n := cmp.Compare(functionImpact(b), functionImpact(a)); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Package, b.Package); n != 0 {
			return n
		}
		if n := cmp.Compare(a.FullName, b.FullName); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Receiver, b.Receiver); n != 0 {
			return n
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return changes, notes
}

func functionImpact(f diffFunction) uint64 {
	var size uint64
	for _, m := range []*byteChange{f.CodeSize, f.PclnBytes, f.ReferencedBytes} {
		if m == nil {
			continue
		}
		delta := m.Delta
		if delta < 0 {
			delta = -delta
		}
		size = max(size, uint64(delta))
	}
	return size
}

func comparePackageFunctions(pkg string, old, next functionInventory, missing map[string]bool) ([]diffFunction, []diagnostic) {
	var changes []diffFunction
	var notes []diagnostic
	oldUsed, newUsed := make([]bool, len(old.functions)), make([]bool, len(next.functions))
	oldKeys, newKeys := map[functionKey][]int{}, map[functionKey][]int{}
	index := func(functions []commonFunction, keys map[functionKey][]int, used []bool) {
		for i, f := range functions {
			if f.Name == "" && f.FullName == "" {
				used[i] = true
				continue
			}
			key := primaryKey(f)
			keys[key] = append(keys[key], i)
		}
	}
	index(old.functions, oldKeys, oldUsed)
	index(next.functions, newKeys, newUsed)
	allKeys := map[functionKey]bool{}
	for k := range oldKeys {
		allKeys[k] = true
	}
	for k := range newKeys {
		allKeys[k] = true
	}
	// Mark ambiguous identities on both sides before matching any other keys.
	ambiguousLegacy := map[functionKey]bool{}
	for key := range allKeys {
		if len(oldKeys[key]) <= 1 && len(newKeys[key]) <= 1 {
			continue
		}
		notes = append(notes, diagnostic{Code: "function_identity_collision", Package: pkg, Identity: key.name, Message: fmt.Sprintf("Function identity has %d old and %d new candidates; ambiguous entries were not compared.", len(oldKeys[key]), len(newKeys[key]))})
		for _, i := range oldKeys[key] {
			oldUsed[i] = true
			ambiguousLegacy[legacyKey(old.functions[i])] = true
		}
		for _, i := range newKeys[key] {
			newUsed[i] = true
			ambiguousLegacy[legacyKey(next.functions[i])] = true
		}
	}
	add := func(o, n *commonFunction) {
		c := makeFunctionChange(pkg, o, n)
		for _, metric := range []struct {
			name  string
			value *byteChange
		}{{"code_size", c.CodeSize}, {"pcln_bytes", c.PclnBytes}, {"referenced_bytes", c.ReferencedBytes}} {
			if metric.value == nil {
				missing[metric.name] = true
			}
		}
		if changedFunction(c) {
			changes = append(changes, c)
		}
	}
	for key, indices := range oldKeys {
		if len(indices) != 1 || len(newKeys[key]) != 1 {
			continue
		}
		i, j := indices[0], newKeys[key][0]
		if oldUsed[i] || newUsed[j] {
			continue
		}
		oldUsed[i], newUsed[j] = true, true
		add(&old.functions[i], &next.functions[j])
	}
	// Protect exact, unambiguous identities before suppressing remaining legacy
	// candidates. A collided generic instance must not hide another full name.
	for i, f := range old.functions {
		if !oldUsed[i] && ambiguousLegacy[legacyKey(f)] {
			oldUsed[i] = true
		}
	}
	for i, f := range next.functions {
		if !newUsed[i] && ambiguousLegacy[legacyKey(f)] {
			newUsed[i] = true
		}
	}
	// A report schema upgrade may add full_name. Fall back only when at least
	// one side lacks it, and the legacy identity has exactly one candidate.
	legacyOld, legacyNew := map[functionKey][]int{}, map[functionKey][]int{}
	for i, f := range old.functions {
		if !oldUsed[i] {
			legacyOld[legacyKey(f)] = append(legacyOld[legacyKey(f)], i)
		}
	}
	for j, f := range next.functions {
		if !newUsed[j] {
			legacyNew[legacyKey(f)] = append(legacyNew[legacyKey(f)], j)
		}
	}
	for key, is := range legacyOld {
		js := legacyNew[key]
		if len(js) == 0 {
			continue
		}
		needsFallback := false
		for _, i := range is {
			needsFallback = needsFallback || old.functions[i].FullName == ""
		}
		for _, j := range js {
			needsFallback = needsFallback || next.functions[j].FullName == ""
		}
		if !needsFallback {
			continue
		}
		if len(is) != 1 || len(js) != 1 {
			notes = append(notes, diagnostic{Code: "function_identity_collision", Package: pkg, Identity: key.name, Message: "Legacy function identity has multiple candidates; schema-upgrade matching is ambiguous and was skipped."})
			for _, i := range is {
				oldUsed[i] = true
			}
			for _, j := range js {
				newUsed[j] = true
			}
			continue
		}
		i, j := is[0], js[0]
		oldUsed[i], newUsed[j] = true, true
		add(&old.functions[i], &next.functions[j])
	}
	if old.complete && next.complete {
		for i := range old.functions {
			if !oldUsed[i] {
				add(&old.functions[i], nil)
			}
		}
		for j := range next.functions {
			if !newUsed[j] {
				add(nil, &next.functions[j])
			}
		}
	}
	slices.SortFunc(notes, func(a, b diagnostic) int { return cmp.Compare(a.Identity, b.Identity) })
	return changes, notes
}
