package internal

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"slices"
)

type KnownAddr struct {
	pclntab entity.AddrSpace

	symbol         entity.AddrSpace
	symbolCoverage []entity.AddrPos

	k *KnownInfo
}

func NewKnownAddr(k *KnownInfo) *KnownAddr {
	return &KnownAddr{
		pclntab: make(map[uint64]*entity.Addr),
		symbol:  make(map[uint64]*entity.Addr),
		k:       k,
	}
}

func (f *KnownAddr) InsertPclntab(entry uint64, size uint64, fn *entity.Function, meta entity.GoPclntabMeta) {
	cur := entity.Addr{
		AddrPos: entity.AddrPos{
			Addr: entry,
			Size: size,
			Type: entity.AddrTypeText,
		},
		Pkg:        fn.Pkg,
		Function:   fn,
		SourceType: entity.AddrSourceGoPclntab,

		Meta: meta,
	}
	f.pclntab.Insert(&cur)
}

func (f *KnownAddr) InsertSymbol(entry uint64, size uint64, p *entity.Package, typ entity.AddrType, meta entity.SymbolMeta) {
	cur := entity.Addr{
		AddrPos: entity.AddrPos{
			Addr: entry,
			Size: size,
			Type: typ,
		},
		Pkg:        p,
		Function:   nil, // TODO: try to find the function?
		SourceType: entity.AddrSourceSymbol,

		Meta: meta,
	}
	if typ == entity.AddrTypeText {
		if _, ok := f.pclntab.Get(entry); ok {
			// pclntab always more accurate
			return
		}
	}
	f.symbol.Insert(&cur)
}

func (f *KnownAddr) BuildSymbolCoverage() {
	f.symbolCoverage = f.symbol.ToCoverage()
}

func (f *KnownAddr) SymbolCovHas(entry uint64, size uint64) (entity.AddrType, bool) {
	c, ok := slices.BinarySearchFunc(f.symbolCoverage, entity.AddrPos{Addr: entry}, func(cur entity.AddrPos, target entity.AddrPos) int {
		if cur.Addr+cur.Size <= target.Addr {
			return -1
		}
		if cur.Addr >= target.Addr+size {
			return 1
		}
		return 0
	})
	if !ok {
		return "", false
	}

	return f.symbolCoverage[c].Type, ok
}

func (f *KnownAddr) InsertDisasm(entry uint64, size uint64, fn *entity.Function) {
	cur := entity.Addr{
		AddrPos: entity.AddrPos{
			Addr: entry,
			Size: size,
			Type: entity.AddrTypeData,
		},
		Pkg:        fn.Pkg,
		Function:   fn,
		SourceType: entity.AddrSourceDisasm,
		Meta:       nil,
	}

	// symbol coverage check
	// this exists since the linker can merge some constant
	typ, ok := f.SymbolCovHas(entry, size)
	if ok {
		if typ != entity.AddrTypeData {
			panic(fmt.Errorf("symbol %x size %x conflict with %s", entry, size, typ))
		}
		// symbol is more accurate
		return
	}

	fn.Disasm.Insert(&cur)
}
