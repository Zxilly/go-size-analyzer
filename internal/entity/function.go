package entity

import (
	"cmp"
	"slices"
)

type FuncType = string

const (
	FuncTypeFunction FuncType = "function"
	FuncTypeMethod   FuncType = "method"
)

type Function struct {
	Name     string    `json:"name"`
	Addr     uint64    `json:"addr"`
	CodeSize uint64    `json:"code_size"`
	Type     FuncType  `json:"type"`
	Receiver string    `json:"receiver"`         // only for methods
	Ranges   []AddrPos `json:"ranges,omitempty"` // non-contiguous DWARF code ranges

	PclnSize     PclnSymbolSize `json:"pcln_size"`
	PclnRanges   []AddrPos      `json:"-"`
	PclnFileSize *uint64        `json:"pcln_bytes,omitempty"`
	Wasm         bool           `json:"-"`
	Source       AddrSourceType `json:"-"`
	pclnBytes    uint64

	disasm AddrSpace
	pkg    *Package
}

func (f *Function) CodeRegions(yield func(AddrPos) bool) {
	if len(f.Ranges) == 0 {
		yield(AddrPos{Addr: f.Addr, Size: f.CodeSize, Type: AddrTypeText})
		return
	}
	for _, r := range f.Ranges {
		if !yield(r) {
			return
		}
	}
}

func (f *Function) Init() {
	f.disasm = make(AddrSpace)
}

func (f *Function) Size() uint64 {
	return f.CodeSize + f.PclnBytes()
}

func (f *Function) PclnBytes() uint64 {
	if f.PclnFileSize != nil {
		return *f.PclnFileSize
	}
	if f.PclnRanges != nil {
		return f.pclnBytes
	}
	return f.PclnSize.Size()
}

func (f *Function) SetPclnRanges(ranges []AddrPos) {
	sorted := slices.Clone(ranges)
	slices.SortFunc(sorted, func(a, b AddrPos) int { return cmp.Compare(a.Addr, b.Addr) })
	merged := sorted[:0]
	for _, r := range sorted {
		if len(merged) > 0 && r.Addr < merged[len(merged)-1].Addr+merged[len(merged)-1].Size {
			last := &merged[len(merged)-1]
			last.Size = max(last.Addr+last.Size, r.Addr+r.Size) - last.Addr
		} else {
			merged = append(merged, r)
		}
	}
	f.PclnRanges = merged
	f.pclnBytes = 0
	var end uint64
	for _, r := range merged {
		if next := r.Addr + r.Size; next > end {
			f.pclnBytes += next - max(end, r.Addr)
			end = next
		}
	}
	n := f.pclnBytes
	f.PclnFileSize = &n
}
