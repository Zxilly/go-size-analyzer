package pkg

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

type RawAddr struct {
	Addr uint64
	Size uint64
}

type AddrWithAddr struct {
	RawAddr
	Pkg *Package
}

type sortedAddr struct {
	values []RawAddr
}

func (s *sortedAddr) Insert(addr RawAddr) {
	i, _ := slices.BinarySearchFunc(s.values, addr, func(i, j RawAddr) int {
		return cmp.Compare(i.Addr, j.Addr)
	})
	s.values = slices.Insert(s.values, i, addr)
}

type FoundAddr struct {
	values map[uint64]*AddrWithAddr
}

func NewFoundAddr() *FoundAddr {
	return &FoundAddr{
		values: make(map[uint64]*AddrWithAddr),
	}
}

var ErrDuplicatePackageForAddr = fmt.Errorf("duplicate package for addr")

// Insert when meet the duplicated addr, keep the larger Size, and return the diff
func (f *FoundAddr) Insert(addr uint64, size uint64, p *Package) error {
	if addrptr, ok := f.values[addr]; ok {
		if addrptr.Pkg.Name != p.Name {
			return errors.Join(ErrDuplicatePackageForAddr, errors.New(addrptr.Pkg.Name), errors.New(p.Name))
		}
		// not overwrite the pclntab info, since it always more accurate
		return nil
	}

	f.values[addr] = &AddrWithAddr{
		RawAddr: RawAddr{
			Addr: addr,
			Size: size,
		},
		Pkg: p,
	}
	return nil
}

func (f *FoundAddr) AssertOverLap() error {
	sa := sortedAddr{
		values: make([]RawAddr, 0, len(f.values)),
	}

	for _, addr := range f.values {
		sa.Insert(RawAddr{Addr: addr.Addr, Size: addr.Size})
	}

	for i := 0; i < len(sa.values)-1; i++ {
		if sa.values[i].Addr+sa.values[i].Size > sa.values[i+1].Addr {
			return fmt.Errorf("addr %x Size %d overlaps addr %x", sa.values[i].Addr, sa.values[i].Size, sa.values[i+1].Addr)
		}
	}
	return nil
}
