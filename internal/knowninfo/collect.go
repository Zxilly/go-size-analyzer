package knowninfo

import (
	"cmp"
	"slices"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

// ValidateNativeCoverage skips recursive coverage construction when address
// types do not overlap. Ambiguous ranges retain the legacy merge's errors and
// disassembly exceptions, which depend on the package hierarchy.
func (k *KnownInfo) ValidateNativeCoverage() error {
	type interval struct {
		start, end uint64
		typ        int
	}
	var bounds []interval
	types := make(map[entity.AddrType]int)
	var count int
	var overflow bool
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		if overflow {
			return nil
		}
		for addr := range p.OwnAddressValues {
			if addr.Size > ^uint64(0)-addr.Addr {
				overflow = true
				return nil
			}
			typ, ok := types[addr.Type]
			if !ok {
				typ = len(bounds)
				types[addr.Type] = typ
				bounds = append(bounds, interval{start: addr.Addr, end: addr.Addr + addr.Size, typ: typ})
			} else {
				bounds[typ].start = min(bounds[typ].start, addr.Addr)
				bounds[typ].end = max(bounds[typ].end, addr.Addr+addr.Size)
			}
			count++
		}
		return nil
	})
	if overflow {
		return k.CollectCoverage()
	}
	compare := func(a, b interval) int {
		if order := cmp.Compare(a.start, b.start); order != 0 {
			return order
		}
		return cmp.Compare(a.end, b.end)
	}
	// Text and data usually occupy separate address envelopes. Proving that
	// separation needs one bound per type, rather than storing and sorting
	// every repeated pclntab range. Overlapping envelopes are inconclusive.
	slices.SortFunc(bounds, compare)
	var end uint64
	var overlap bool
	for _, bound := range bounds {
		overlap = overlap || bound.start < end
		end = max(end, bound.end)
	}
	if !overlap {
		return nil
	}
	intervals := make([]interval, 0, count)
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		for addr := range p.OwnAddressValues {
			intervals = append(intervals, interval{start: addr.Addr, end: addr.Addr + addr.Size, typ: types[addr.Type]})
		}
		return nil
	})
	slices.SortFunc(intervals, compare)
	furthest := make([]uint64, len(types))
	for _, current := range intervals {
		// Keep empty ranges: the legacy merge can reject an empty range
		// whose start is inside a range of another type.
		for typ, end := range furthest {
			if typ != current.typ && current.start < end {
				return k.CollectCoverage()
			}
		}
		furthest[current.typ] = max(furthest[current.typ], current.end)
	}
	return nil
}

func (k *KnownInfo) CollectCoverage() error {
	covs := make([]entity.AddrCoverage, 0, len(k.Deps.TopPkgs))
	for _, pkg := range k.Deps.TopPkgs {
		cov, err := pkg.GetPackageCoverage()
		if err != nil {
			return err
		}
		covs = append(covs, cov)
	}
	var err error
	k.Coverage, err = entity.MergeAndCleanCoverage(covs)
	return err
}

func (k *KnownInfo) CalculateSectionSize() error {
	for _, s := range k.Sects.Sections {
		s.KnownSize = 0
	}
	for _, part := range k.Coverage {
		start, end := part.Pos.Addr, part.Pos.Addr+part.Pos.Size
		for _, s := range k.Sects.Sections {
			if s.Debug || s.OnlyInMemory || s.VirtualSection || s.ContentType == entity.SectionContentOther {
				continue
			}
			lo, hi := max(start, s.Addr), min(end, s.Addr+min(s.Size, s.FileSize))
			if lo < hi {
				s.KnownSize += hi - lo
			}
		}
	}
	return nil
}

func (k *KnownInfo) CalculatePackageSize() error {
	if w, ok := k.Wrapper.(*wrapper.WasmWrapper); ok {
		for fn := range k.Deps.Functions {
			size := w.FileDataIntervals(fn.PclnRanges)
			fn.PclnFileSize = &size
		}
	}
	return k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		if w, ok := k.Wrapper.(*wrapper.WasmWrapper); ok {
			cov, err := p.GetPackageCoverage()
			if err != nil {
				return err
			}
			p.Size = w.FileDataSize(cov)
			var code []entity.FileRange
			var collect func(*entity.Package)
			collect = func(pkg *entity.Package) {
				for fn := range pkg.Functions {
					if r, ok := w.FunctionFileRange(fn.Addr, k.VersionFlag.Meq125); ok {
						code = append(code, r)
					}
				}
				for _, sub := range pkg.SubPackages {
					collect(sub)
				}
			}
			collect(p)
			p.Size += entity.UnionFileSize(code)
			return nil
		}
		return p.AssignPackageSize()
	})
}
