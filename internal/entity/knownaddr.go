package entity

import (
	"cmp"
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

const typeDebug = false

func (f *KnownAddr) cancelIfSectionTypeMismatch(cur *Addr, as AddrSpace) bool {
	if !f.sect.IsType(cur.Addr, cur.Size, cur.Type) {
		if typeDebug {
			sect := f.sect.FindSection(cur.Addr, cur.Size)
			name := "unknown"
			if sect != nil {
				name = sect.Name
			}

			slog.Debug(fmt.Sprintf("section type mismatch addr: %s belongs to %s", cur, name))
		}
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
	// SymbolCovHas binary-searches; the slice must be sorted.
	f.SymbolCoverage = f.SymbolAddrSpace.ToDirtyCoverage()
	slices.SortFunc(f.SymbolCoverage, compareCoveragePart)
}

func (f *KnownAddr) SymbolCovHas(entry uint64, size uint64) (AddrType, bool) {
	if len(f.SymbolCoverage) == 0 {
		return "", false
	}

	end := entry + size
	// Symbols are non-overlapping and sorted, so only the immediate predecessor
	// of the first entry with Addr >= end can overlap [entry, end).
	idx, _ := slices.BinarySearchFunc(f.SymbolCoverage, end, func(cur *CoveragePart, target uint64) int {
		return cmp.Compare(cur.Pos.Addr, target)
	})
	if idx > 0 {
		prev := f.SymbolCoverage[idx-1]
		if prev.Pos.Addr+prev.Pos.Size > entry {
			return prev.Pos.Type, true
		}
	}

	return "", false
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

	// Linker may place non-data values (function pointers in type descriptors,
	// itabs, etc.) whose symbols overlap disasm candidates — drop as false
	// positives rather than treating them as real strings.
	typ, ok := f.SymbolCovHas(entry, size)
	if ok {
		if typ != AddrTypeData {
			slog.Debug(fmt.Sprintf("disasm addr %x size %x overlaps symbol of type %s, dropped", entry, size, typ))
		}
		return
	}

	f.cancelIfSectionTypeMismatch(&cur, fn.disasm)
}
