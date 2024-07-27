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
