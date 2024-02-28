package pkg

import (
	"cmp"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
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

type sizeOnce struct {
	sync.Once
	val uint64
}

type AddrCandidates struct {
	typ  AddrType
	addr uint64

	pclntabAddr *Addr
	symbolAddr  *Addr
	disasmAddrs []*Addr

	size sizeOnce
}

func (p *AddrCandidates) Size() uint64 {
	p.size.Do(func() {
		switch p.typ {
		case AddrTypeText:
			// try to load from pclntab first
			if p.pclntabAddr != nil {
				p.size.val = p.pclntabAddr.Size
				return
			}
			// then symbol
			if p.symbolAddr != nil {
				p.size.val = p.symbolAddr.Size
				return
			}
			panic(fmt.Errorf("no size for %s", p))
		case AddrTypeData:
			// try to load from the largest disasm
			if len(p.disasmAddrs) > 0 {
				size := uint64(0)
				for _, d := range p.disasmAddrs {
					size = max(size, d.Size)
				}
				p.size.val = size
				return
			}
			// if any symbol exists, use it
			if p.symbolAddr != nil {
				p.size.val = p.symbolAddr.Size
				return
			}
			panic(fmt.Errorf("no size for %s", p))
		default:
			panic(fmt.Errorf("unknown type: %s", p.typ))
		}
	})
	return p.size.val
}

func (p *AddrCandidates) String() string {
	msg := fmt.Sprintf("Addr: %x Type: %s\n", p.addr, p.typ)
	msg += fmt.Sprintf("Pclntab: %s\n", p.pclntabAddr)
	msg += fmt.Sprintf("Symbol: %s\n", p.symbolAddr)
	for _, d := range p.disasmAddrs {
		msg += fmt.Sprintf("Disasm: %s\n", d)
	}
	return msg
}

func (p *AddrCandidates) Add(addr *Addr) error {
	if p.addr != addr.Addr {
		return errors.New("addr not match")
	}

	switch addr.Source {
	case AddrSourceGoPclntab:
		if p.pclntabAddr != nil {
			return errors.New("pclntab already exist")
		}
		p.pclntabAddr = addr
	case AddrSourceSymbol:
		if p.symbolAddr != nil {
			return errors.New("symbol already exist")
		}
		p.symbolAddr = addr
	case AddrSourceDisasm:
		for i, d := range p.disasmAddrs {
			// perform a simple check
			// FIXME: handle bytes
			oldMeta := d.Meta.(DisasmMeta)
			newMeta := addr.Meta.(DisasmMeta)

			oldContent := oldMeta.DisasmString
			newContent := newMeta.DisasmString

			// one of them is the prefix of the other
			if len(oldContent) > len(newContent) {
				if !strings.HasPrefix(oldContent, newContent) {
					return fmt.Errorf("disasm content not match: %s %s", oldContent, newContent)
				}
			} else {
				if !strings.HasPrefix(newContent, oldContent) {
					return fmt.Errorf("disasm content not match: %s %s", oldContent, newContent)
				}
			}

			if d.Pkg.Name == addr.Pkg.Name {
				// store the largest one by replacing the old one
				if d.Size < addr.Size {
					p.disasmAddrs[i] = addr
				}
				return nil
			}
		}
		p.disasmAddrs = append(p.disasmAddrs, addr)
	}

	return nil
}

func NewAddrCandidates(addr *Addr) *AddrCandidates {
	ret := &AddrCandidates{
		typ:  addr.Type,
		addr: addr.Addr,
		size: sizeOnce{Once: sync.Once{}},
	}
	_ = ret.Add(addr)
	return ret
}

type KnownAddr struct {
	memory map[uint64]*AddrCandidates
}

func NewFoundAddr() *KnownAddr {
	return &KnownAddr{
		memory: make(map[uint64]*AddrCandidates),
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
		f.memory[addr] = NewAddrCandidates(&cur)
		return
	}
	err := pkgOnAddr.Add(&cur)
	if err != nil {
		log.Fatalf("Add addr failed: %s %v", err, cur)
	}
}

// ValidateOverlap check addr overlap
func (f *KnownAddr) ValidateOverlap() error {
	addrs := make([]*AddrCandidates, 0)
	for _, pkgsOnAddr := range f.memory {
		addrs = append(addrs, pkgsOnAddr)
	}

	slices.SortFunc(addrs, func(a, b *AddrCandidates) int {
		return cmp.Compare(a.addr, b.addr)
	})

	for i := 0; i < len(addrs)-1; i++ {
		cur := addrs[i]
		next := addrs[i+1]

		// This made me curious, pclntab size always greater 1 than the symbol size
		// which one is the correct one?
		// FIXME: handle this
		if cur.addr+cur.Size() <= next.addr {
			continue
		}

		// overlap happened
		if !(cur.typ == AddrTypeData && next.typ == AddrTypeData) {
			return fmt.Errorf("overlap only allowed in rodata:\n"+
				"%s\n%s", cur, next)
		}

		// we believe the overlap was caused by linker optimization
		continue
	}

	return nil
}

func (f *KnownAddr) RemoveDuplicate() error {

}
