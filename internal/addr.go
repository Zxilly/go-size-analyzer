package internal

import (
	"cmp"
	"fmt"

	"slices"
)

type AddrType string

const (
	AddrTypeUnknown AddrType = "unknown" // it exists, but should never be collected
	AddrTypeText             = "text"    // for text section
	AddrTypeData             = "data"    // data / rodata section
)

type AddrPos struct {
	Addr uint64
	Size uint64
}

type Addr struct {
	AddrPos

	Pkg      *Package
	Function *Function // for symbol source it will be a nil

	Source AddrSourceType
	Type   AddrType

	Meta any
}

func (a Addr) String() string {
	msg := fmt.Sprintf("Addr: %x Size: %x Pkg: %s Source: %s", a.Addr, a.Size, a.Pkg.Name, a.Source)
	msg += fmt.Sprintf(" Meta: %#v", a.Meta)
	return msg
}

type AddrCoverage []AddrPos

type AddrSpace map[uint64]*Addr

func (a AddrSpace) Get(addr uint64) (ret *Addr, ok bool) {
	ret, ok = a[addr]
	return
}

func (a AddrSpace) Insert(addr *Addr) {
	old, ok := a.Get(addr.Addr)
	if ok {
		// use the larger one
		if old.Size < addr.Size {
			a[addr.Addr] = addr
		}
		return
	}
	a[addr.Addr] = addr
}

func (a AddrSpace) Coverage() AddrCoverage {
	ranges := make([]*Addr, 0)
	for _, addr := range a {
		ranges = append(ranges, addr)
	}

	slices.SortFunc(ranges, func(a, b *Addr) int {
		if a.Addr != b.Addr {
			return cmp.Compare(a.Addr, b.Addr)
		}
		return cmp.Compare(a.Size, b.Size)
	})

	cover := make([]AddrPos, 0)
	var last *Addr
	for _, r := range ranges {
		if len(cover) == 0 {
			cover = append(cover, r.AddrPos)
			last = r
			continue
		}

		if last.Addr+last.Size >= r.Addr {
			// merge
			if last.Type != r.Type {
				panic(fmt.Errorf("addr %x type %s and %s conflict", r.Addr, last.Type, r.Type))
			}

			if last.Addr+last.Size < r.Addr+r.Size {
				last.Size = r.Addr + r.Size - last.Addr
			}
		} else {
			cover = append(cover, r.AddrPos)
		}
		last = r
	}

	return cover
}

type KnownAddr struct {
	pclntab AddrSpace

	symbol         AddrSpace // package can be nil for cgo symbols
	symbolCoverage []AddrPos

	k *KnownInfo
}

func NewKnownAddr(k *KnownInfo) *KnownAddr {
	return &KnownAddr{
		pclntab: make(map[uint64]*Addr),
		symbol:  make(map[uint64]*Addr),
		k:       k,
	}
}

func (f *KnownAddr) InsertPclntab(addr uint64, size uint64, fn *Function, meta GoPclntabMeta) {
	cur := Addr{
		AddrPos: AddrPos{
			Addr: addr,
			Size: size,
		},
		Pkg:      fn.Pkg,
		Function: fn,
		Source:   AddrSourceGoPclntab,
		Type:     AddrTypeText,
		Meta:     meta,
	}
	f.pclntab.Insert(&cur)
}

func (f *KnownAddr) InsertSymbol(addr uint64, size uint64, p *Package, typ AddrType, meta SymbolMeta) {
	cur := Addr{
		AddrPos: AddrPos{
			Addr: addr,
			Size: size,
		},
		Pkg:      p,
		Function: nil, // TODO: try to find the function?
		Source:   AddrSourceSymbol,
		Type:     typ,
		Meta:     meta,
	}
	if typ == AddrTypeText {
		if _, ok := f.pclntab.Get(addr); ok {
			// pclntab always more accurate
			return
		}
	}
	f.symbol.Insert(&cur)
}

func (f *KnownAddr) BuildSymbolCoverage() {
	f.symbolCoverage = f.symbol.Coverage()
}

func (f *KnownAddr) SymbolCovHas(addr uint64, size uint64) bool {
	_, ok := slices.BinarySearchFunc(f.symbolCoverage, AddrPos{Addr: addr}, func(cur AddrPos, target AddrPos) int {
		if cur.Addr+cur.Size <= target.Addr {
			return -1
		}
		if cur.Addr >= target.Addr+size {
			return 1
		}
		return 0
	})
	return ok
}

func (f *KnownAddr) InsertDisasm(addr uint64, size uint64, fn *Function, meta DisasmMeta) {
	cur := Addr{
		AddrPos: AddrPos{
			Addr: addr,
			Size: size,
		},
		Pkg:      fn.Pkg,
		Function: fn,
		Source:   AddrSourceDisasm,
		Type:     AddrTypeData,
		Meta:     meta,
	}

	// symbol type check
	if sv, ok := f.symbol.Get(addr); ok {
		if sv.Type != AddrTypeData {
			panic(fmt.Errorf("addr %x already in symbol, but not data type", addr))
		}
		// symbol is more accurate
		return
	}
	// symbol coverage check
	// this exists since the linker can merge some constant
	if f.SymbolCovHas(addr, size) {
		// symbol coverage is more accurate
		return
	}

	fn.Disasm.Insert(&cur)
}
