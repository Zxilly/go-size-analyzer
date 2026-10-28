package entity

import (
	"fmt"
	"strings"
)

type AddrPos struct {
	Addr uint64
	Size uint64
	Type AddrType
}

func (a *AddrPos) String() string {
	return fmt.Sprintf("Addr: 0x%x CodeSize: %d Type: %s", a.Addr, a.Size, a.Type)
}

type Addr struct {
	*AddrPos

	Pkg *Package // package can be nil for cgo symbols

	Function *Function // for symbol source it will be a nil
	Symbol   *Symbol   // for function source it will be a nil

	SourceType AddrSourceType

	// Role distinguishes a declaration from a package's use of existing data.
	// An empty role in older callers is treated as a definition.
	Role     string
	Boundary string

	// evidence is a flat set of immutable address snapshots. Only AddrSpace
	// representatives carry it; the snapshots never point back to this slice.
	evidence []*Addr
}

// AddrValue is a read-only projection of one address claim. Its position is a
// value so synchronous accounting does not allocate a synthetic Addr/AddrPos
// pair for every pclntab range. It never contains an AddrSpace's other claims.
type AddrValue struct {
	AddrPos
	Pkg        *Package
	Function   *Function
	Symbol     *Symbol
	SourceType AddrSourceType
	Role       string
	Boundary   string
}

// Keep each address and its position in one allocation. A snapshot owns its
// position even when the source came from a different space or a caller's stack.
type addrSnapshot struct {
	addr Addr
	pos  AddrPos
}

func (s *addrSnapshot) set(a *Addr) *Addr {
	s.addr = *a
	s.pos = *a.AddrPos
	s.addr.AddrPos = &s.pos
	s.addr.evidence = nil
	if s.addr.Role == "" {
		s.addr.Role = AddrRoleDefinition
	}
	return &s.addr
}

func (a *Addr) snapshot() *Addr {
	return new(addrSnapshot).set(a)
}

func (a *Addr) setSourceEvidence() {
	a.Role, a.Boundary = sourceEvidence(a.SourceType, a.Boundary)
}

func sourceEvidence(source AddrSourceType, boundary string) (role, resolvedBoundary string) {
	switch source {
	case AddrSourceSymbol:
		return AddrRoleDefinition, "symbol"
	case AddrSourceDwarf:
		return AddrRoleDefinition, "declared"
	case AddrSourceGoPclntab, AddrSourceTyp:
		return AddrRoleDefinition, "encoded"
	case AddrSourceStaticRead, AddrSourceStaticCopy:
		return AddrRoleReference, "instruction"
	case AddrSourceDisasm, AddrSourceStaticHeader:
		return AddrRoleReference, "heuristic"
	default:
		// Unspecified sources retain the caller-provided boundary.
		return AddrRoleDefinition, boundary
	}
}

func (a *Addr) String() string {
	ret := new(strings.Builder)
	_, _ = fmt.Fprintf(ret, "AddrPos: %s", a.AddrPos)
	if a.Pkg != nil {
		_, _ = fmt.Fprintf(ret, " Pkg: %s", a.Pkg.Name)
	}
	if a.Function != nil {
		_, _ = fmt.Fprintf(ret, " Function: %s", a.Function.Name)
	}
	if a.Symbol != nil {
		_, _ = fmt.Fprintf(ret, " Symbol: %s", a.Symbol.Name)
	}
	_, _ = fmt.Fprintf(ret, " SourceType: %s", a.SourceType)
	return ret.String()
}
