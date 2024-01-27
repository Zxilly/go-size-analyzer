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
	values := make([]Addr, 0, len(f.values))
	for _, v := range f.values {
		values = append(values, v)
	}
	slices.SortFunc(values, func(i, j Addr) int {
		return cmp.Compare(i.Addr, j.Addr)
	})

	for i := 0; ; i++ {
		first := values[i]
		second := values[i+1]

		if first.Addr+first.Size > second.Addr {
			// is that a rodata overwritten?
			if first.Addr+first.Size >= second.Addr+second.Size {
				// yes, it's a rodata overwritten, remove second
				// thanks clever linker :P
				delete(f.values, second.Addr)
				values = append(values[:i+1], values[i+2:]...)
				continue
			}

			return fmt.Errorf(
				"{addr %x Size:%d Pass:%s Pkg:%s Meta:%#v}"+
					" overlaps "+
					"{addr %x Size:%d Pass:%s Pkg:%s Meta:%#v}",
				first.Addr, first.Size, first.Pass, first.Pkg.Name, first.Meta,
				second.Addr, second.Size, second.Pass, second.Pkg.Name, second.Meta,
			)
		}

		if i+2 >= len(values) {
			break
		}
	}
	return nil
}
