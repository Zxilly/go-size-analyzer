package entity

import (
	"fmt"
	"log/slog"
	"slices"
)

type KnownAddr struct {
	TextAddrSpace AddrSpace

	SymbolAddrSpace AddrSpace
	SymbolCoverage  AddrCoverage

	sect *Store
}

func NewKnownAddr(sect *Store) *KnownAddr {
	return &KnownAddr{
		TextAddrSpace:   make(map[uint64]*Addr),
		SymbolAddrSpace: make(map[uint64]*Addr),
		SymbolCoverage:  make(AddrCoverage, 0),
		sect:            sect,
	}
}

func (f *KnownAddr) cancelIfSectionTypeMismatch(cur *Addr, as AddrSpace) bool {
	if !f.sect.IsType(cur.Addr, cur.Size, cur.Type) {
		sect := f.sect.FindSection(cur.Addr, cur.Size)
		name := "unknown"
		if sect != nil {
			name = sect.Name
		}

		slog.Debug(fmt.Sprintf("section type mismatch addr: %s belongs to %s", cur, name))
		return false
	}
	as.Insert(cur)
	return true
}

func (f *KnownAddr) InsertTextFromPclnTab(entry uint64, size uint64, fn *Function) {
	cur := Addr{
		AddrPos: &AddrPos{
			Addr: entry,
			Size: size,
			Type: AddrTypeText,
		},
		Pkg:        fn.pkg,
		Function:   fn,
		SourceType: AddrSourceGoPclntab,
	}
	f.cancelIfSectionTypeMismatch(&cur, f.TextAddrSpace)
}

func (f *KnownAddr) InsertTextFromDWARF(entry uint64, size uint64, fn *Function) {
	cur := Addr{
		AddrPos: &AddrPos{
			Addr: entry,
			Size: size,
			Type: AddrTypeText,
		},
		Pkg:        fn.pkg,
		Function:   fn,
		SourceType: AddrSourceDwarf,
	}
	f.cancelIfSectionTypeMismatch(&cur, f.TextAddrSpace)
}

func (f *KnownAddr) InsertSymbol(symbol *Symbol, p *Package) *Addr {
	cur := &Addr{
		AddrPos: &AddrPos{
			Addr: symbol.Addr,
			Size: symbol.Size,
			Type: symbol.Type,
		},
		Pkg:        p,
		Function:   nil,
		Symbol:     symbol,
		SourceType: AddrSourceSymbol,
	}

	ok := f.cancelIfSectionTypeMismatch(cur, f.SymbolAddrSpace)
	if !ok {
		return nil
	}

	return cur
}

func (f *KnownAddr) InsertSymbolFromDWARF(symbol *Symbol, p *Package) *Addr {
	cur := &Addr{
		AddrPos: &AddrPos{
			Addr: symbol.Addr,
			Size: symbol.Size,
			Type: symbol.Type,
		},
		Pkg:        p,
		Function:   nil,
		Symbol:     symbol,
		SourceType: AddrSourceDwarf,
	}
	ok := f.cancelIfSectionTypeMismatch(cur, f.SymbolAddrSpace)
	if !ok {
		return nil
	}
	return cur
}

func (f *KnownAddr) BuildSymbolCoverage() {
	f.SymbolCoverage = f.SymbolAddrSpace.ToDirtyCoverage()
}

func (f *KnownAddr) SymbolCovHas(entry uint64, size uint64) (AddrType, bool) {
	if len(f.SymbolCoverage) == 0 {
		return "", false
	}

	c, ok := slices.BinarySearchFunc(f.SymbolCoverage, &CoveragePart{Pos: &AddrPos{Addr: entry}}, func(cur *CoveragePart, target *CoveragePart) int {
		if cur.Pos.Addr+cur.Pos.Size <= target.Pos.Addr {
			return -1
		}
		if cur.Pos.Addr >= target.Pos.Addr+size {
			return 1
		}
		return 0
	})
	if !ok {
		return "", false
	}

	return f.SymbolCoverage[c].Pos.Type, ok
}

func (f *KnownAddr) InsertDisasm(entry uint64, size uint64, fn *Function) {
	cur := Addr{
		AddrPos: &AddrPos{
			Addr: entry,
			Size: size,
			Type: AddrTypeData,
		},
		Pkg:        fn.pkg,
		Function:   fn,
		SourceType: AddrSourceDisasm,
	}

	// symbol coverage check
	// this exists since the linker can merge some constant
	typ, ok := f.SymbolCovHas(entry, size)
	if ok {
		if typ != AddrTypeData {
			panic(fmt.Errorf("symbol %x size %x conflict with %s", entry, size, typ))
		}
		// symbol is more accurate
		return
	}

	f.cancelIfSectionTypeMismatch(&cur, fn.disasm)
}
