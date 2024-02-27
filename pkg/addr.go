package pkg

import (
	"fmt"
	"log"
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

	Source AddrSource
	Type   AddrType

	Meta any
}

func (a Addr) String() string {
	msg := fmt.Sprintf("Addr: %x Size: %x Pkg: %s Source: %s", a.Addr, a.Size, a.Pkg.Name, a.Source)
	msg += fmt.Sprintf(" Meta: %#v", a.Meta)
	return msg
}

type PackageAddrs = map[AddrSource]Addr

type PackagesOnAddr = map[string]PackageAddrs

func NewPackageAddr() PackagesOnAddr {
	return make(PackagesOnAddr)
}

type KnownAddr struct {
	values map[uint64]PackagesOnAddr
}

func NewFoundAddr() *KnownAddr {
	return &KnownAddr{
		values: make(map[uint64]PackagesOnAddr),
	}
}

func (f *KnownAddr) Insert(addr uint64, size uint64, p *Package, src AddrSource, typ AddrType, meta any) {
	cur := Addr{
		Addr:   addr,
		Size:   size,
		Pkg:    p,
		Source: src,
		Type:   typ,
		Meta:   meta,
	}

	// Did we already have this addr recorded?
	if _, ok := f.values[addr]; !ok {
		f.values[addr] = NewPackageAddr()
	}

	// Did we already have this src at this addr?
	if _, ok := f.values[addr][p.Name]; !ok {
		f.values[addr][p.Name] = make(PackageAddrs)
	}

	pkgAddrs := f.values[addr][p.Name]

	// determine if we should overwrite the old value
	// or just ignore the new one
	if old, ok := pkgAddrs[src]; !ok {
		// never recorded, just insert
		pkgAddrs[src] = cur
		return
	} else {
		if src != AddrSourceDisasm {
			// unrecoverable error
			log.Fatalf("Addr %x already recorded, old:%s new:%s", addr, old, cur)
		} else {
			// disasm can have multiple results for the same string, use the longer one
			if cur.Size > old.Size {
				pkgAddrs[src] = cur
			}
		}
	}
}

func (f *KnownAddr) Validate() error {
	return nil
}
