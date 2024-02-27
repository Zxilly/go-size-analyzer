package pkg

import (
	"errors"
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

	Source AddrSourceType
	Type   AddrType

	Meta any
}

func (a Addr) String() string {
	msg := fmt.Sprintf("Addr: %x Size: %x Pkg: %s Source: %s", a.Addr, a.Size, a.Pkg.Name, a.Source)
	msg += fmt.Sprintf(" Meta: %#v", a.Meta)
	return msg
}

type PackageAddrs struct {
	sources map[AddrSourceType]*Addr
}

func (p *PackageAddrs) Insert(addr *Addr) error {
	cur, ok := p.sources[addr.Source]
	if !ok {
		// don't have this source, add it
		p.sources[addr.Source] = addr
		return nil
	}

	switch addr.Source {
	case AddrSourceGoPclntab, AddrSourceSymbol:
		// should not duplicate
		return errors.New("duplicate source")
	case AddrSourceDisasm:
		// always keep the largest size
		if addr.Size > cur.Size {
			p.sources[addr.Source] = addr
		}
		return nil
	default:
		panic("unreachable")
	}
}

func NewPackageAddrs(addr *Addr) *PackageAddrs {
	return &PackageAddrs{
		sources: map[AddrSourceType]*Addr{
			addr.Source: addr,
		},
	}
}

// PackagesOnAddr package name -> types
type PackagesOnAddr struct {
	pkgs map[string]*PackageAddrs
	typ  AddrType
}

func (p *PackagesOnAddr) Insert(addr *Addr) error {
	if addr.Type != p.typ {
		return errors.New("type mismatch")
	}

	sources, ok := p.pkgs[addr.Pkg.Name]
	if !ok {
		p.pkgs[addr.Pkg.Name] = NewPackageAddrs(addr)
		return nil
	}
	return sources.Insert(addr)
}

func NewPackagesOnAddr(addr *Addr) *PackagesOnAddr {
	return &PackagesOnAddr{
		pkgs: map[string]*PackageAddrs{
			addr.Pkg.Name: NewPackageAddrs(addr),
		},
		typ: addr.Type,
	}
}

type KnownAddr struct {
	memory map[uint64]*PackagesOnAddr
}

func NewFoundAddr() *KnownAddr {
	return &KnownAddr{
		memory: make(map[uint64]*PackagesOnAddr),
	}
}

func (f *KnownAddr) Insert(addr uint64, size uint64, p *Package, src AddrSourceType, typ AddrType, meta any) {
	cur := Addr{
		Addr:   addr,
		Size:   size,
		Pkg:    p,
		Source: src,
		Type:   typ,
		Meta:   meta,
	}

	pkgOnAddr, ok := f.memory[addr]
	if !ok {
		f.memory[addr] = NewPackagesOnAddr(&cur)
		return
	}
	err := pkgOnAddr.Insert(&cur)
	if err != nil {
		log.Fatalf("Insert addr failed: %s %v", err, cur)
	}
}

func (f *KnownAddr) Validate() error {
	return nil
}
