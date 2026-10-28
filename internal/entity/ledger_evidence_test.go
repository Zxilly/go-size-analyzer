package entity_test

import (
	"slices"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestFileLedgerSeparatesDefinitionsAndReferrers(t *testing.T) {
	claims := []entity.FileClaim{
		{FileRange: entity.FileRange{Offset: 0, Size: 10}, Class: 1, Source: "header"},
		{FileRange: entity.FileRange{Offset: 10, Size: 40}, Class: 2, Owner: "a", Source: "symbol", Boundary: "symbol"},
		{FileRange: entity.FileRange{Offset: 10, Size: 20}, Class: 2, Owner: "a", Source: "dwarf", Boundary: "declared"},
		{FileRange: entity.FileRange{Offset: 20, Size: 20}, Class: 2, Owner: "b", Source: "static_read", Role: "reference", Boundary: "instruction"},
		{FileRange: entity.FileRange{Offset: 60, Size: 20}, Class: 2, Owner: "c", Source: "static_copy", Role: "reference"},
		{FileRange: entity.FileRange{Offset: 70, Size: 20}, Class: 2, Owner: "d", Source: "static_copy", Role: "reference"},
	}
	build := func(claims []entity.FileClaim, details bool) *entity.FileCoverage {
		ledger := entity.NewFileLedger(100)
		for _, claim := range claims {
			require.NoError(t, ledger.Add(claim))
		}
		return ledger.Finish(details)
	}
	coverage := build(claims, true)
	require.Equal(t, uint64(70), coverage.Attributed)
	require.Equal(t, uint64(10), coverage.Recognized)
	require.Equal(t, uint64(20), coverage.Unclassified)
	require.Equal(t, uint64(10), coverage.Shared, "a reader does not become a second definition owner")
	require.Equal(t, map[string]uint64{"a": 40, "c": 20, "d": 20}, coverage.PackageSizes)
	require.Equal(t, map[string]uint64{"symbol": 40, "dwarf": 20, "static_read": 20, "static_copy": 30}, coverage.BySource)
	for _, region := range coverage.Regions {
		if region.Offset == 20 {
			require.Equal(t, uint64(10), region.Size)
			require.Equal(t, []string{"a"}, region.Owners)
			require.Equal(t, []string{"a"}, region.DefinitionOwners)
			require.Equal(t, []string{"b"}, region.Referrers)
			require.Len(t, region.Evidence, 3)
		}
		if region.Offset == 30 {
			require.Equal(t, []string{"static_read", "symbol"}, region.Sources, "DWARF's shorter span must not be extended")
		}
	}
	slices.Reverse(claims)
	require.Equal(t, coverage, build(claims, true))
	claims = append(claims, claims[0], claims[2])
	require.Equal(t, coverage, build(claims, true))
	withoutDetails := build(claims, false)
	coverage.Regions = nil
	require.Equal(t, coverage, withoutDetails)
}

func TestFileLedgerProjectsMetadataAndPackageGroupsByUnion(t *testing.T) {
	ledger := entity.NewFileLedger(30)
	ledger.SetPackageGroups(map[string][]string{
		"p": {"p"}, "p/child": {"p", "p/child"},
	})
	for _, claim := range []entity.FileClaim{
		{FileRange: entity.FileRange{Offset: 0, Size: 20}, Class: 2, Owner: "p", Source: "symbol_table", Metadata: true},
		{FileRange: entity.FileRange{Offset: 10, Size: 10}, Class: 2, Owner: "p/child", Source: "symbol_name", Metadata: true},
		{FileRange: entity.FileRange{Offset: 20, Size: 10}, Class: 2, Owner: "p/child", Source: "pclntab"},
	} {
		require.NoError(t, ledger.Add(claim))
	}
	coverage := ledger.Finish(true)
	require.Equal(t, uint64(30), coverage.Attributed)
	require.Equal(t, map[string]uint64{"p": 30, "p/child": 20}, coverage.PackageSizes)
	require.Equal(t, map[string]uint64{"p": 20, "p/child": 10}, coverage.MetadataSizes)
	require.Equal(t, []entity.FileRange{{Offset: 0, Size: 30}}, coverage.AttributedRanges)
}

func TestFileLedgerRejectsUnknownAttributionRole(t *testing.T) {
	ledger := entity.NewFileLedger(10)
	require.Error(t, ledger.Add(entity.FileClaim{FileRange: entity.FileRange{Size: 5}, Class: 2, Role: "guess"}))
	require.Equal(t, uint64(10), ledger.Finish(false).Unclassified)
}
