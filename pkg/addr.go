package pkg

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

type AddrParsePass int

const (
	AddrPassGoPclntab AddrParsePass = iota
	AddrPassSymbol
	AddrPassDisasm
)

func (p AddrParsePass) String() string {
	switch p {
	case AddrPassGoPclntab:
		return "pclntab"
	case AddrPassSymbol:
		return "symbol"
	case AddrPassDisasm:
		return "disasm"
	default:
		return "unknown"
	}
}

type Addr struct {
	Addr uint64
	Size uint64
	Pkg  *Package

	Pass AddrParsePass
	// for debug only
	Meta string
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

var ErrDuplicatePackageForAddr = fmt.Errorf("duplicate package for addr")

func (f *FoundAddr) Insert(addr uint64, size uint64, p *Package, pass AddrParsePass, meta string) error {
	if addrptr, ok := f.values[addr]; ok {
		if addrptr.Pkg.Name != p.Name {
			return errors.Join(ErrDuplicatePackageForAddr, errors.New(addrptr.Pkg.Name), errors.New(p.Name))
		}
		// not overwrite the pclntab info, since it always more accurate
		return nil
	}

	f.values[addr] = Addr{
		Addr: addr,
		Size: size,
		Pkg:  p,
		Pass: pass,
		Meta: meta,
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

	for i := 0; i < len(sa.values)-1; i++ {
		if sa.values[i].Addr+sa.values[i].Size > sa.values[i+1].Addr {
			return fmt.Errorf(
				"addr {%x Size:%d Pass:%s Pkg:%s Meta:%s}"+
					" overlaps "+
					"{addr %x Size:%d Pass:%s Pkg:%s Meta:%s}",
				sa.values[i].Addr, sa.values[i].Size, sa.values[i].Pass, sa.values[i].Pkg.Name, sa.values[i].Meta,
				sa.values[i+1].Addr, sa.values[i+1].Size, sa.values[i+1].Pass, sa.values[i+1].Pkg.Name, sa.values[i+1].Meta,
			)
		}
	}
	return nil
}
