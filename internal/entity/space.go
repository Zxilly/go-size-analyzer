package entity

import (
	"cmp"
	"slices"
)

// AddrSpace keeps a preferred lookup at each starting address, together with
// every distinct claim at that address. Get remains a constant-time lookup;
// All and ToDirtyCoverage preserve the original boundaries of all evidence.
type AddrSpace map[uint64]*Addr

func (a AddrSpace) Get(addr uint64) (ret *Addr, ok bool) {
	ret, ok = a[addr]
	return ret, ok
}

func (a AddrSpace) Insert(addr *Addr) {
	if addr == nil || addr.AddrPos == nil {
		return
	}
	old, exists := a.Get(addr.Addr)
	if !exists && len(addr.evidence) == 0 {
		a[addr.Addr] = addr.snapshot()
		return
	}
	if exists && len(addr.evidence) == 0 {
		a.insertSingle(addr, old)
		return
	}
	var claims []*Addr
	if exists {
		claims = appendEvidence(claims, old)
	}
	claims = appendEvidence(claims, addr)
	slices.SortFunc(claims, compareAddrEvidence)
	claims = slices.CompactFunc(claims, func(a, b *Addr) bool {
		return compareAddrIdentity(a, b) == 0
	})
	best := claims[0]
	for _, claim := range claims[1:] {
		if preferAddr(claim, best) {
			best = claim
		}
	}
	stored := best.snapshot()
	if len(claims) > 1 {
		stored.evidence = claims
	}
	a[addr.Addr] = stored
}

// Single claims dominate collection. Their insertion point is already known
// from the sorted identity set, so neither sorting nor duplicate compaction is
// needed. Changed sets still own independent snapshots of all their claims.
func (a AddrSpace) insertSingle(addr, old *Addr) {
	previous := old.evidence
	if len(previous) == 0 {
		if compareAddrIdentity(old, addr) == 0 {
			if compareAddrEntity(old, addr) > 0 {
				a[addr.Addr] = addr.snapshot()
			}
			return
		}
		previous = []*Addr{old}
	}
	index, found := slices.BinarySearchFunc(previous, addr, compareAddrIdentity)
	if found && compareAddrEntity(previous[index], addr) <= 0 {
		return
	}
	count := len(previous)
	if !found {
		count++
	}
	claims := make([]*Addr, count)
	snapshots := make([]addrSnapshot, count+1)
	var best *Addr
	for i := range claims {
		source := addr
		if i != index {
			oldIndex := i
			if !found && i > index {
				oldIndex--
			}
			source = previous[oldIndex]
		}
		claims[i] = snapshots[i].set(source)
		if best == nil || preferAddr(claims[i], best) {
			best = claims[i]
		}
	}
	stored := snapshots[count].set(best)
	stored.evidence = claims
	a[addr.Addr] = stored
}

func appendEvidence(dst []*Addr, addr *Addr) []*Addr {
	if len(addr.evidence) == 0 {
		return append(dst, addr.snapshot())
	}
	for _, claim := range addr.evidence {
		dst = append(dst, claim.snapshot())
	}
	return dst
}

func addrPackageName(addr *Addr) string {
	if addr.Pkg != nil {
		return addr.Pkg.Name
	}
	return ""
}

func addrEntityName(addr *Addr) string {
	if addr.Function != nil {
		return addr.Function.Name
	}
	if addr.Symbol != nil {
		return addr.Symbol.Name
	}
	return ""
}

func addrEntityKind(addr *Addr) int {
	if addr.Function != nil {
		return 2
	}
	if addr.Symbol != nil {
		return 1
	}
	return 0
}

func addrFunctionEntry(addr *Addr) uint64 {
	if addr.Function != nil {
		return addr.Function.Addr
	}
	return 0
}

func addrRole(addr *Addr) string {
	if addr.Role == "" {
		return AddrRoleDefinition
	}
	return addr.Role
}

// A claim's identity describes its extent and provenance. Several symbols or
// functions in one package can provide the same evidence for shared data.
func compareAddrIdentity(a, b *Addr) int {
	if n := cmp.Compare(a.Addr, b.Addr); n != 0 {
		return n
	}
	if n := cmp.Compare(a.Size, b.Size); n != 0 {
		return n
	}
	if n := cmp.Compare(a.Type, b.Type); n != 0 {
		return n
	}
	if n := cmp.Compare(addrPackageName(a), addrPackageName(b)); n != 0 {
		return n
	}
	if n := cmp.Compare(a.SourceType, b.SourceType); n != 0 {
		return n
	}
	if n := cmp.Compare(addrRole(a), addrRole(b)); n != 0 {
		return n
	}
	return cmp.Compare(a.Boundary, b.Boundary)
}

func compareAddrEntity(a, b *Addr) int {
	if n := cmp.Compare(addrEntityKind(a), addrEntityKind(b)); n != 0 {
		return n
	}
	if n := cmp.Compare(addrEntityName(a), addrEntityName(b)); n != 0 {
		return n
	}
	return cmp.Compare(addrFunctionEntry(a), addrFunctionEntry(b))
}

// Entity names choose a deterministic representative of equivalent claims;
// pointer identity never affects deduplication or ordering.
func compareAddrEvidence(a, b *Addr) int {
	if order := compareAddrIdentity(a, b); order != 0 {
		return order
	}
	return compareAddrEntity(a, b)
}

func boundaryRank(boundary string) int {
	switch boundary {
	case "encoded", "declared":
		return 4
	case "symbol", "instruction":
		return 3
	case "heuristic":
		return 1
	case "estimated":
		return 0
	default:
		return 2
	}
}

func sourceRank(source AddrSourceType) int {
	switch source {
	case AddrSourceDwarf:
		return 4
	case AddrSourceTyp:
		return 3
	case AddrSourceGoPclntab:
		return 2
	case AddrSourceSymbol:
		return 1
	default:
		return 0
	}
}

func preferAddr(a, b *Addr) bool {
	definition := func(addr *Addr) bool { return addr.Role != AddrRoleReference }
	if definition(a) != definition(b) {
		return definition(a)
	}
	if ar, br := boundaryRank(a.Boundary), boundaryRank(b.Boundary); ar != br {
		return ar > br
	}
	if ar, br := sourceRank(a.SourceType), sourceRank(b.SourceType); ar != br {
		return ar > br
	}
	if a.Size != b.Size {
		return a.Size > b.Size
	}
	return compareAddrEvidence(a, b) < 0
}

// All visits evidence in deterministic order. The returned records retain
// their individual sizes; a larger claim never widens a smaller one.
func (a AddrSpace) All(yield func(*Addr) bool) {
	if len(a) <= 1 {
		for _, addr := range a {
			if len(addr.evidence) == 0 {
				yield(addr)
				return
			}
			for _, claim := range addr.evidence {
				if !yield(claim) {
					return
				}
			}
			return
		}
		return
	}
	starts := make([]uint64, 0, len(a))
	for start := range a {
		starts = append(starts, start)
	}
	slices.Sort(starts)
	for _, start := range starts {
		addr := a[start]
		if len(addr.evidence) == 0 {
			if !yield(addr) {
				return
			}
			continue
		}
		for _, claim := range addr.evidence {
			if !yield(claim) {
				return
			}
		}
	}
}

func MergeAddrSpace(others ...AddrSpace) AddrSpace {
	ret := make(AddrSpace)
	for _, other := range others {
		for _, addr := range other {
			ret.Insert(addr)
		}
	}
	return ret
}

// ToDirtyCoverage returns every unmerged evidence range.
func (a AddrSpace) ToDirtyCoverage() AddrCoverage {
	coverage := make(AddrCoverage, 0, len(a))
	for addr := range a.All {
		pos := *addr.AddrPos
		coverage = append(coverage, &CoveragePart{Pos: &pos, Addrs: []*Addr{addr}})
	}
	return coverage
}
