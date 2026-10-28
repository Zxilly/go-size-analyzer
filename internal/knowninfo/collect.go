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
	var intervals []interval
	types := make(map[entity.AddrType]int)
	var furthest []uint64
	var overflow bool
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		if overflow {
			return nil
		}
		for addr := range p.OwnAddresses {
			if addr.Size > ^uint64(0)-addr.Addr {
				overflow = true
				return nil
			}
			typ, ok := types[addr.Type]
			if !ok {
				typ = len(furthest)
				types[addr.Type] = typ
				furthest = append(furthest, 0)
			}
			intervals = append(intervals, interval{start: addr.Addr, end: addr.Addr + addr.Size, typ: typ})
		}
		return nil
	})
	if overflow {
		return k.CollectCoverage()
	}
	slices.SortFunc(intervals, func(a, b interval) int {
		return cmp.Or(cmp.Compare(a.start, b.start), cmp.Compare(a.end, b.end))
	})
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
