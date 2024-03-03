package pkg

import (
	"fmt"
)

type AddrType string

const (
	AddrTypeUnknown AddrType = "unknown"
	AddrTypeText             = "text" // for text section
	AddrTypeData             = "data" // data / rodata section
)

type Addr struct {
	Addr uint64
	Size uint64
	Pkg  *Package

	Source AddrSourceType
	Type   AddrType

	Meta any
}

func (a Addr) String() string {
	msg := fmt.Sprintf("Addr: %x Size: %x Pkg: %s Source: %s", a.Addr, a.Size, a.Pkg.Name, a.Source)
	msg += fmt.Sprintf(" Meta: %#v", a.Meta)
	return msg
}

type KnownAddr struct {
	pclntab map[uint64]*Addr
	symbol  map[uint64]*Addr            // package can be nil for cgo symbols
	disasms map[uint64]map[string]*Addr // each package can have one disasm result
}

func NewFoundAddr() *KnownAddr {
	return &KnownAddr{
		pclntab: make(map[uint64]*Addr),
		symbol:  make(map[uint64]*Addr),
		disasms: make(map[uint64]map[string]*Addr),
	}
}

func (f *KnownAddr) InsertPclntab(addr uint64, size uint64, p *Package, meta GoPclntabMeta) {
	cur := Addr{
		Addr:   addr,
		Size:   size,
		Pkg:    p,
		Source: AddrSourceGoPclntab,
		Type:   AddrTypeText,
		Meta:   meta,
	}
	f.pclntab[addr] = &cur
}

func (f *KnownAddr) InsertSymbol(addr uint64, size uint64, p *Package, typ AddrType, meta SymbolMeta) {
	cur := Addr{
		Addr:   addr,
		Size:   size,
		Pkg:    p,
		Source: AddrSourceSymbol,
		Type:   typ,
		Meta:   meta,
	}
	if typ == AddrTypeText {
		if _, ok := f.pclntab[addr]; ok {
			// pclntab always more accurate
			return
		}
	}
	f.symbol[addr] = &cur
}

func (f *KnownAddr) InsertDisasm(addr uint64, size uint64, p *Package, meta DisasmMeta) {
	cur := Addr{
		Addr:   addr,
		Size:   size,
		Pkg:    p,
		Source: AddrSourceDisasm,
		Type:   AddrTypeData,
		Meta:   meta,
	}

	if sv, ok := f.symbol[addr]; ok {
		if sv.Type != AddrTypeData {
			panic(fmt.Sprintf("addr %x already in symbol, but not data type", addr))
		}
		// symbol is more accurate
		return
	}

	if addrs, ok := f.disasms[addr]; !ok {
		f.disasms[addr] = map[string]*Addr{p.Name: &cur}
	} else {
		old, ok := addrs[p.Name]
		if ok {
			// keep the larger one
			if old.Size < size {
				addrs[p.Name] = &cur
			}
		} else {
			// just store
			addrs[p.Name] = &cur
		}
	}
}
