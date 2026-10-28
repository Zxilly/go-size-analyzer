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
	symbolPrefix    []int // index of the furthest-reaching symbol in each prefix

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
	f.insertTextEvidence(&cur)
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
	f.insertTextEvidence(&cur)
}

func (f *KnownAddr) InsertTextFromSymbol(entry, size uint64, fn *Function) {
	cur := &Addr{AddrPos: &AddrPos{Addr: entry, Size: size, Type: AddrTypeText}, Pkg: fn.pkg, Function: fn, SourceType: AddrSourceSymbol}
	f.insertTextEvidence(cur)
}

func (f *KnownAddr) insertTextEvidence(cur *Addr) {
	cur.setSourceEvidence()
	if f.cancelIfSectionTypeMismatch(cur, f.TextAddrSpace) {
		if cur.Function.codeEvidence == nil {
			cur.Function.codeEvidence = AddrSpace{}
		}
		cur.Function.codeEvidence.Insert(cur)
	}
}

func (f *KnownAddr) InsertSymbol(symbol *Symbol, p *Package, sources ...AddrSourceType) *Addr {
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

	if len(sources) > 0 {
		cur.SourceType = sources[0]
	}
	cur.setSourceEvidence()
	if symbol.Role != "" {
		cur.Role = symbol.Role
	}
	if symbol.Boundary != "" {
		cur.Boundary = symbol.Boundary
	}
	ok := f.cancelIfSectionTypeMismatch(cur, f.SymbolAddrSpace)
	if !ok {
		return nil
	}

	return cur
}

func (f *KnownAddr) InsertSymbolFromDWARF(symbol *Symbol, p *Package) *Addr {
	return f.InsertSymbol(symbol, p, AddrSourceDwarf)
}

func (f *KnownAddr) BuildSymbolCoverage() {
	// SymbolCovHas binary-searches; the slice must be sorted.
	f.SymbolCoverage = f.SymbolAddrSpace.ToDirtyCoverage()
	slices.SortFunc(f.SymbolCoverage, compareCoveragePart)
	f.symbolPrefix = make([]int, len(f.SymbolCoverage))
	longest := 0
	for i, part := range f.SymbolCoverage {
		if previous := f.SymbolCoverage[longest].Pos; part.Pos.Addr+part.Pos.Size > previous.Addr+previous.Size {
			longest = i
		}
		f.symbolPrefix[i] = longest
	}
}

func (f *KnownAddr) SymbolCovHas(entry uint64, size uint64) (AddrType, bool) {
	if len(f.SymbolCoverage) == 0 || size == 0 || size > ^uint64(0)-entry {
		return "", false
	}

	end := entry + size
	// Aliases can sit inside larger symbols. The furthest-reaching prefix
	// handles nesting without scanning backwards through every smaller alias.
	idx, _ := slices.BinarySearchFunc(f.SymbolCoverage, end, func(cur *CoveragePart, target uint64) int {
		return cmp.Compare(cur.Pos.Addr, target)
	})
	if idx > 0 {
		prev := f.SymbolCoverage[f.symbolPrefix[idx-1]]
		if prev.Pos.Addr+prev.Pos.Size > entry {
			return prev.Pos.Type, true
		}
	}

	return "", false
}

func (f *KnownAddr) InsertDisasm(entry uint64, size uint64, fn *Function, sources ...AddrSourceType) {
	if size == 0 || size > ^uint64(0)-entry {
		return
	}
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

	if len(sources) > 0 {
		cur.SourceType = sources[0]
	}
	cur.setSourceEvidence()
	// Instruction-sized reads and copies may legitimately reference an existing
	// data object. String heuristics still reject overlapping symbols, where
	// pointer-shaped metadata could otherwise be mistaken for string contents.
	typ, ok := f.SymbolCovHas(entry, size)
	exactAccess := cur.SourceType == AddrSourceStaticCopy || cur.SourceType == AddrSourceStaticRead
	if ok && (typ != AddrTypeData || !exactAccess) {
		if typ != AddrTypeData {
			slog.Debug(fmt.Sprintf("disasm addr %x size %x overlaps symbol of type %s, dropped", entry, size, typ))
		}
		return
	}

	f.cancelIfSectionTypeMismatch(&cur, fn.disasm)
}
