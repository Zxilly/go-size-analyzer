package knowninfo

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

func (k *KnownInfo) CollectCoverage() error {
	covs := make([]entity.AddrCoverage, 0, len(k.Deps.TopPkgs))
	for _, pkg := range k.Deps.TopPkgs {
		covs = append(covs, pkg.GetPackageCoverage())
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

func (k *KnownInfo) CalculatePackageSize() {
	if w, ok := k.Wrapper.(*wrapper.WasmWrapper); ok {
		for fn := range k.Deps.Functions {
			size := w.FileDataIntervals(fn.PclnRanges)
			fn.PclnFileSize = &size
		}
	}
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		if w, ok := k.Wrapper.(*wrapper.WasmWrapper); ok {
			p.Size = w.FileDataSize(p.GetPackageCoverage())
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
		p.AssignPackageSize()
		return nil
	})
}
