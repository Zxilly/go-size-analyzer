package entity

import (
	"fmt"
)

type Addr struct {
	AddrPos

	Pkg      *Package  // package can be nil for cgo symbols
	Function *Function // for symbol source it will be a nil

	SourceType AddrSourceType

	Meta any
}

func (a Addr) String() string {
	msg := fmt.Sprintf("Addr: %x Size: %x pkg: %s SourceType: %s", a.Addr, a.Size, a.Pkg.Name, a.SourceType)
	msg += fmt.Sprintf(" Meta: %#v", a.Meta)
	return msg
}
