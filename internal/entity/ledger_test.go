package entity_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestFileLedgerPartitionsSharedAndStandaloneBytes(t *testing.T) {
	claims := []entity.FileClaim{
		{FileRange: entity.FileRange{Offset: 0, Size: 10}, Class: 1, Source: "header"},
		{FileRange: entity.FileRange{Offset: 10, Size: 40}, Class: 2, Owner: "a", Source: "symbol"},
		{FileRange: entity.FileRange{Offset: 30, Size: 30}, Class: 2, Owner: "b", Source: "dwarf"},
		{FileRange: entity.FileRange{Offset: 20, Size: 20}, Class: 1, Source: "metadata"},
	}
	build := func(claims []entity.FileClaim) *entity.FileCoverage {
		l := entity.NewFileLedger(100)
		for _, c := range claims {
			require.NoError(t, l.Add(c))
		}
		return l.Finish(true)
	}
	r := build(claims)
	require.Equal(t, uint64(50), r.Attributed)
	require.Equal(t, uint64(10), r.Recognized)
	require.Equal(t, uint64(40), r.Unclassified)
	require.Equal(t, uint64(20), r.Shared)
	var offset uint64
	for _, region := range r.Regions {
		require.Equal(t, offset, region.Offset)
		offset += region.Size
	}
	require.Equal(t, uint64(100), offset)
	slices.Reverse(claims)
	require.Equal(t, r, build(claims))
	claims = append(claims, claims[2])
	require.Equal(t, r, build(claims))
}

func TestFileLedgerRejectsOutOfFileAndOverflowRanges(t *testing.T) {
	l := entity.NewFileLedger(100)
	require.Error(t, l.Add(entity.FileClaim{FileRange: entity.FileRange{Offset: 99, Size: 2}}))
	require.Error(t, l.Add(entity.FileClaim{FileRange: entity.FileRange{Offset: math.MaxUint64, Size: 2}}))
	require.Equal(t, uint64(100), l.Finish(false).Unclassified)
}
