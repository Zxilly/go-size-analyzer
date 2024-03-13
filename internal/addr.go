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

type AddrCov struct {
	Addr uint64
	Size uint64
}

func AddrCovCmp(a, b AddrCov) int {
	if a.Addr != b.Addr {
		return cmp.Compare(a.Addr, b.Addr)
	}
	return cmp.Compare(a.Size, b.Size)
}

type Addr struct {
	AddrCov

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

type DisasmAddrs map[uint64]*Addr

func NewDisasmAddrs() DisasmAddrs {
	return make(DisasmAddrs)
}

func (a DisasmAddrs) Add(addr *Addr) {
	if pre, ok := a[addr.Addr]; ok {
		// keep the larger one
		if pre.Size < addr.Size {
			a[addr.Addr] = addr
		}
	} else {
		a[addr.Addr] = addr
	}
}

type KnownAddr struct {
	pclntab map[uint64]*Addr

	symbol         map[uint64]*Addr // package can be nil for cgo symbols
	symbolCoverage []AddrCov

	disasms map[string]DisasmAddrs // each package has their own addr space
}

func NewFoundAddr() *KnownAddr {
	return &KnownAddr{
		pclntab: make(map[uint64]*Addr),
		symbol:  make(map[uint64]*Addr),
		disasms: make(map[string]DisasmAddrs),
	}
}

func (f *KnownAddr) InsertPclntab(addr uint64, size uint64, fn *Function, meta GoPclntabMeta) {
	cur := Addr{
		AddrCov: AddrCov{
			Addr: addr,
			Size: size,
		},
		Pkg:      fn.Pkg,
		Function: fn,
		Source:   AddrSourceGoPclntab,
		Type:     AddrTypeText,
		Meta:     meta,
	}
	f.pclntab[addr] = &cur
}

func (f *KnownAddr) InsertSymbol(addr uint64, size uint64, p *Package, typ AddrType, meta SymbolMeta) {
	cur := Addr{
		AddrCov: AddrCov{
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
		if _, ok := f.pclntab[addr]; ok {
			// pclntab always more accurate
			return
		}
	}
	f.symbol[addr] = &cur
}

func (f *KnownAddr) BuildSymbolCoverage() {
	ranges := make([]AddrCov, 0)
	for _, addr := range f.symbol {
		if addr.Type != AddrTypeData {
			continue
		}

		ranges = append(ranges, AddrCov{
			Addr: addr.Addr,
			Size: addr.Size,
		})
	}

	slices.SortFunc(ranges, AddrCovCmp)

	cover := make([]AddrCov, 0)
	for _, r := range ranges {
		if len(cover) == 0 {
			cover = append(cover, r)
			continue
		}
		last := cover[len(cover)-1]
		if last.Addr+last.Size >= r.Addr {
			// merge
			if last.Addr+last.Size < r.Addr+r.Size {
				last.Size = r.Addr + r.Size - last.Addr
			}
		} else {
			cover = append(cover, r)
		}
	}
	f.symbolCoverage = cover
}

func (f *KnownAddr) SymbolCovHas(addr uint64, size uint64) bool {
	_, ok := slices.BinarySearchFunc(f.symbolCoverage, AddrCov{Addr: addr}, func(cur AddrCov, target AddrCov) int {
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
		AddrCov: AddrCov{
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
	if sv, ok := f.symbol[addr]; ok {
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

	// create a new disasm addr space if not exists
	if _, ok := f.disasms[fn.Pkg.Name]; !ok {
		f.disasms[fn.Pkg.Name] = NewDisasmAddrs()
	}

	f.disasms[fn.Pkg.Name].Add(&cur)
}
