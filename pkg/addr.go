package pkg

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

type Addr struct {
	Addr uint64
	Size uint64
	Pkg  *Package

	Pass AddrParsePass
	// for debug only
	Meta any
}

type sortedAddr struct {
	values []Addr
}

func (s *sortedAddr) Insert(addr Addr) {
	i, _ := slices.BinarySearchFunc(s.values, addr, func(i, j Addr) int {
		return cmp.Compare(i.Addr, j.Addr)
	})
	s.values = slices.Insert(s.values, i, addr)
}

type FoundAddr struct {
	values map[uint64]Addr
}

func NewFoundAddr() *FoundAddr {
	return &FoundAddr{
		values: make(map[uint64]Addr),
	}
}

var ErrDifferentPackageForAddr = errors.New("different package for addr")

func (f *FoundAddr) Insert(addr uint64, size uint64, p *Package, pass AddrParsePass, meta InternMeta) error {
	if addrPtr, ok := f.values[addr]; ok {
		if addrPtr.Pkg.Name != p.Name {
			return errors.Join(ErrDifferentPackageForAddr,
				fmt.Errorf("previous known: %#v %s", addrPtr.Meta, addrPtr.Pass),
				fmt.Errorf("current: %#v %s", meta, pass),
			)
		}
		// not overwrite the previous info
		return nil
	}

	f.values[addr] = Addr{
		Addr: addr,
		Size: size,
		Pkg:  p,
		Pass: pass,
		Meta: meta.GetInternedMeta(),
	}
	return nil
}

func (f *FoundAddr) AssertOverLap() error {
	sa := sortedAddr{
		values: make([]Addr, 0, len(f.values)),
	}

	for _, addr := range f.values {
		sa.Insert(addr)
	}

	// handle the case that link optimization merge rodata

	for i := 0; i < len(sa.values)-1; i++ {
		first := sa.values[i]
		second := sa.values[i+1]

		if first.Addr+first.Size > second.Addr {
			return fmt.Errorf(
				"{addr %x Size:%d Pass:%s Pkg:%s Meta:%#v}"+
					" overlaps "+
					"{addr %x Size:%d Pass:%s Pkg:%s Meta:%#v}",
				first.Addr, first.Size, first.Pass, first.Pkg.Name, first.Meta,
				second.Addr, second.Size, second.Pass, second.Pkg.Name, second.Meta,
			)
		}
	}
	return nil
}
