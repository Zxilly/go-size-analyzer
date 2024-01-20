package go_size_view

import (
	"cmp"
	"fmt"
	"slices"
)

type Addr struct {
	Addr uint64
	Size uint64
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
	values map[uint64]uint64
}

func NewFoundAddr() *FoundAddr {
	return &FoundAddr{
		values: make(map[uint64]uint64),
	}
}

// Insert when meet the duplicated addr, keep the larger size, and return the diff
func (f *FoundAddr) Insert(addr uint64, size uint64) uint64 {
	if _, ok := f.values[addr]; ok {
		if f.values[addr] >= size {
			return 0
		} else {
			diff := size - f.values[addr]
			f.values[addr] = size
			return diff
		}
	}

	f.values[addr] = size
	return size
}

func (f *FoundAddr) AssertOverLap() error {
	sa := sortedAddr{
		values: make([]Addr, 0, len(f.values)),
	}

	for addr, size := range f.values {
		sa.Insert(Addr{Addr: addr, Size: size})
	}

	for i := 0; i < len(sa.values)-1; i++ {
		if sa.values[i].Addr+sa.values[i].Size > sa.values[i+1].Addr {
			return fmt.Errorf("addr %x size %d overlaps addr %x", sa.values[i].Addr, sa.values[i].Size, sa.values[i+1].Addr)
		}
	}
	return nil
}
