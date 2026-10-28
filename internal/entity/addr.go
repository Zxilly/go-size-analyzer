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

func (a *Addr) snapshot() *Addr {
	cloned := *a
	pos := *a.AddrPos
	cloned.AddrPos = &pos
	cloned.evidence = nil
	if cloned.Role == "" {
		cloned.Role = AddrRoleDefinition
	}
	return &cloned
}

func (a *Addr) setSourceEvidence() {
	a.Role = AddrRoleDefinition
	switch a.SourceType {
	case AddrSourceSymbol:
		a.Boundary = "symbol"
	case AddrSourceDwarf:
		a.Boundary = "declared"
	case AddrSourceGoPclntab, AddrSourceTyp:
		a.Boundary = "encoded"
	case AddrSourceStaticRead, AddrSourceStaticCopy:
		a.Role, a.Boundary = AddrRoleReference, "instruction"
	case AddrSourceDisasm, AddrSourceStaticHeader:
		a.Role, a.Boundary = AddrRoleReference, "heuristic"
	default:
		// Unspecified sources retain the caller-provided boundary.
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
