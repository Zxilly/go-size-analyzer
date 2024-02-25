package pkg

import (
	"fmt"
	"log"
)

type Addr struct {
	Addr uint64
	Size uint64
	Pkg  *Package

	Pass AddrParseType
	// for debug only
	Meta any
}

func (a Addr) String() string {
	msg := fmt.Sprintf("Addr: %x Size: %x Pkg: %s Pass: %s", a.Addr, a.Size, a.Pkg.Name, a.Pass)
	msg += fmt.Sprintf(" Meta: %#v", a.Meta)
	return msg
}

type PackagePassAddr = map[AddrParseType]Addr

type PackageAddr = map[string]PackagePassAddr

func NewPackageAddr() PackageAddr {
	return make(PackageAddr)
}

type FoundAddr struct {
	values map[uint64]PackageAddr
}

func NewFoundAddr() *FoundAddr {
	return &FoundAddr{
		values: make(map[uint64]PackageAddr),
	}
}

func (f *FoundAddr) Insert(addr uint64, size uint64, p *Package, pass AddrParseType, meta any) {
	cur := Addr{
		Addr: addr,
		Size: size,
		Pkg:  p,
		Pass: pass,
		Meta: meta,
	}

	// Did we already have this addr recorded?
	if _, ok := f.values[addr]; !ok {
		f.values[addr] = NewPackageAddr()
	}

	// Did we already have this pass at this addr?
	if _, ok := f.values[addr][p.Name]; !ok {
		f.values[addr][p.Name] = make(PackagePassAddr)
	}

	// determine if we should overwrite the old value
	// or just ignore the new one
	if old, ok := f.values[addr][p.Name][pass]; !ok {
		// never recorded, just insert
		f.values[addr][p.Name][pass] = cur
	} else {
		if pass != AddrPassDisasm {
			// unrecoverable error
			log.Fatalf("Addr %x already recorded, old:%s new:%s", addr, old, cur)
		} else {
			// disasm can have multiple results, use the longer one
			if cur.Size > old.Size {
				f.values[addr][p.Name][pass] = cur
			}
		}
	}
}

func (f *FoundAddr) Validate() error {
	return nil
}
