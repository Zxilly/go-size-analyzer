package entity

import (
	"cmp"
	"maps"
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
		match := old
		if len(old.evidence) > 0 {
			index, found := slices.BinarySearchFunc(old.evidence, addr, compareAddrIdentity)
			if found {
				match = old.evidence[index]
			} else {
				match = nil
			}
		}
		// Shared names and package paths are commonly reported once per type.
		// Repeated equivalent provenance must not copy the growing claim set.
		if match != nil && compareAddrIdentity(match, addr) == 0 && compareAddrEntity(match, addr) <= 0 {
			return
		}
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
	return cmp.Or(
		cmp.Compare(a.Addr, b.Addr),
		cmp.Compare(a.Size, b.Size),
		cmp.Compare(a.Type, b.Type),
		cmp.Compare(addrPackageName(a), addrPackageName(b)),
		cmp.Compare(a.SourceType, b.SourceType),
		cmp.Compare(addrRole(a), addrRole(b)),
		cmp.Compare(a.Boundary, b.Boundary),
	)
}

func compareAddrEntity(a, b *Addr) int {
	return cmp.Or(
		cmp.Compare(addrEntityKind(a), addrEntityKind(b)),
		cmp.Compare(addrEntityName(a), addrEntityName(b)),
		cmp.Compare(addrFunctionEntry(a), addrFunctionEntry(b)),
	)
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
	for _, start := range slices.Sorted(maps.Keys(a)) {
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
