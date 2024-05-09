package entity

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

type Symbol struct {
	Name string   `json:"name"`
	Addr uint64   `json:"addr"`
	Size uint64   `json:"size"`
	Type AddrType `json:"type"`
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
