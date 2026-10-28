package knowninfo

import (
	"slices"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

type metadataTarget struct {
	addr uint64
	typ  entity.AddrType
}

type metadataOwner struct {
	pkg, name string
}

// attributeSymbolMetadata uses already established definitions. A symbol name
// by itself cannot create a new owner, and a reference to an object does not
// acquire its symbol table entry. File ranges never enter the VA address map.
func (k *KnownInfo) attributeSymbolMetadata(ledger *entity.FileLedger, records []wrapper.SymbolMetadata) error {
	wanted := make(map[metadataTarget][]metadataOwner, len(records))
	for _, record := range records {
		wanted[metadataTarget{record.Addr, record.Type}] = nil
	}
	if len(wanted) == 0 {
		return nil
	}
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		for addr := range p.OwnAddresses {
			key := metadataTarget{addr.Addr, addr.Type}
			owners, ok := wanted[key]
			if !ok || addr.Size == 0 || addr.Role == "reference" {
				continue
			}
			var name string
			if addr.Symbol != nil {
				name = addr.Symbol.Name
			} else if addr.Function != nil && addr.Type == entity.AddrTypeText {
				name = addr.Function.Name
			} else {
				// Per-function pclntab spans do not define native data symbols.
				continue
			}
			owner := metadataOwner{pkg: p.Name, name: name}
			if !slices.Contains(owners, owner) {
				wanted[key] = append(owners, owner)
			}
		}
		return nil
	})
	for _, record := range records {
		candidates := wanted[metadataTarget{record.Addr, record.Type}]
		matched := make([]metadataOwner, 0, len(candidates))
		for _, candidate := range candidates {
			if record.Name == candidate.name || record.Name == candidate.pkg+"."+candidate.name {
				matched = append(matched, candidate)
			}
		}
		if len(matched) == 0 {
			// Equivalent evidence can retain only one alias name. A different
			// name is safe to use only when every same-address definition has
			// the same package; ambiguous records remain recognized metadata.
			if len(candidates) == 0 || slices.ContainsFunc(candidates[1:], func(candidate metadataOwner) bool {
				return candidate.pkg != candidates[0].pkg
			}) {
				continue
			}
			matched = candidates[:1]
		}
		seen := make(map[string]bool, len(matched))
		for _, owner := range matched {
			if seen[owner.pkg] {
				continue
			}
			seen[owner.pkg] = true
			for _, r := range record.Ranges {
				if err := ledger.Add(entity.FileClaim{FileRange: r.FileRange, Class: 2, Owner: owner.pkg, Source: r.Source, Role: "definition", Boundary: "encoded", Metadata: true}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (k *KnownInfo) packageGroups() map[string][]string {
	groups := make(map[string][]string)
	var visit func(*entity.Package, []string)
	visit = func(p *entity.Package, parents []string) {
		ancestors := append(slices.Clone(parents), p.Name)
		groups[p.Name] = ancestors
		for _, sub := range p.SubPackages {
			visit(sub, ancestors)
		}
	}
	for _, p := range k.Deps.TopPkgs {
		visit(p, nil)
	}
	return groups
}

// Package and section totals are projections of the same unique physical
// partition. This includes file-only metadata and excludes BSS and suppressed
// reference ownership without adding virtual offsets to physical offsets.
func (k *KnownInfo) applyFileAccounting(coverage *entity.FileCoverage, sections []*entity.Section) {
	_ = k.Deps.Trie.Walk(func(_ string, p *entity.Package) error {
		p.Size = coverage.PackageSizes[p.Name]
		p.MetadataSize = coverage.MetadataSizes[p.Name]
		return nil
	})
	for _, section := range sections {
		section.KnownSize = 0
		if section.OnlyInMemory || section.VirtualSection || section.FileSize == 0 {
			continue
		}
		start, end := section.Offset, section.Offset+section.FileSize
		index, _ := slices.BinarySearchFunc(coverage.AttributedRanges, start, func(r entity.FileRange, offset uint64) int {
			if r.Offset+r.Size <= offset {
				return -1
			}
			return 1
		})
		for _, r := range coverage.AttributedRanges[index:] {
			if r.Offset >= end {
				break
			}
			section.KnownSize += min(r.Offset+r.Size, end) - max(r.Offset, start)
		}
	}
}
