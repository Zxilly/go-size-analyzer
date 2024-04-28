package entity

import (
	"github.com/samber/lo"
)

// AddrSpace is a map of address to Addr
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

func MergeAddrSpace(others ...AddrSpace) AddrSpace {
	ret := make(AddrSpace)
	for _, other := range others {
		for _, addr := range other {
			ret.Insert(addr)
		}
	}
	return ret
}

// ToDirtyCoverage get the coverage of the current address space
func (a AddrSpace) ToDirtyCoverage() AddrCoverage {
	return lo.MapToSlice(a, func(k uint64, v *Addr) *CoveragePart {
		return &CoveragePart{
			Pos:   v.AddrPos,
			Addrs: []*Addr{v},
		}
	})
}

func (a AddrSpace) ToCleanCoverage() (AddrCoverage, error) {
	return CleanCoverage(a.ToDirtyCoverage())
}
