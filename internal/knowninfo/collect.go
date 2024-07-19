package knowninfo

import (
	"fmt"
	"log/slog"
	"math"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (k *KnownInfo) CollectCoverage() error {
	// load coverage for pclntab and symbol
	pclntabCov := k.KnownAddr.TextAddrSpace.ToDirtyCoverage()

	// merge all
	covs := make([]entity.AddrCoverage, 0)

	// collect packages coverage
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		covs = append(covs, p.GetPackageCoverage())
		return nil
	})

	covs = append(covs, pclntabCov, k.KnownAddr.SymbolCoverage)

	var err error
	k.Coverage, err = entity.MergeAndCleanCoverage(covs)
	return err
}

func (k *KnownInfo) CalculateSectionSize() error {
	sectCache := make(map[*entity.Section]uint64)
	// minus coverage part
	for _, cp := range k.Coverage {
		s := k.Sects.FindSection(cp.Pos.Addr, cp.Pos.Size)
		if s == nil {
			slog.Debug(fmt.Sprintf("possible bss addr %s", cp))
			continue
		}
		sectCache[s] += cp.Pos.Size
	}

	pclntabSize := uint64(0)
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		for fn := range p.Functions {
			pclntabSize += fn.PclnSize.Size()
		}
		return nil
	})

	// minus pclntab size
	var pclntabSection *entity.Section
	for _, s := range k.Sects.Sections {
		if s.Addr <= k.PClnTabAddr && k.PClnTabAddr < s.AddrEnd {
			pclntabSection = s
			sectCache[s] += pclntabSize
			break
		}
	}
	if pclntabSection == nil {
		slog.Warn(fmt.Sprintf("pclntab addr %d not in any section", k.PClnTabAddr))
	}

	// linear map virtual size to file size
	for s, size := range sectCache {
		mapper := 1.0
		if s.Size != s.FileSize {
			// need to map to file size
			mapper = float64(s.FileSize) / float64(s.Size)
		}
		s.KnownSize = uint64(math.Floor(float64(size) * mapper))

		if s.KnownSize > s.FileSize && s.FileSize != 0 {
			if s != pclntabSection {
				// pclntab contains lots of reuse
				// todo: add coverage support for pclntab size
				slog.Warn(fmt.Sprintf("section %s known size %d > file size %d", s.Name, s.KnownSize, s.FileSize))
			}
			s.KnownSize = s.FileSize
		}

		if s.FileSize == 0 {
			s.KnownSize = 0
		}
	}
	return nil
}

// CalculatePackageSize calculate the size of each package
// Happens after disassembly
func (k *KnownInfo) CalculatePackageSize() {
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		p.AssignPackageSize()
		return nil
	})
}
