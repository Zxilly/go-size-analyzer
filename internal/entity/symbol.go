package entity

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

type Symbol struct {
	Name string
	Addr uint64
	Size uint64
	Type AddrType
}

func NewSymbol(name string, addr, size uint64, typ AddrType) *Symbol {
	return &Symbol{
		Name: utils.Deduplicate(name),
		Addr: addr,
		Size: size,
		Type: typ,
	}
}

func (s *Symbol) String() string {
	return fmt.Sprintf("Symbol: %s Addr: %x Size: %x Type: %s", s.Name, s.Addr, s.Size, s.Type)
}
