package entity_test

import (
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func conflictingPackage() *entity.Package {
	p := entity.NewPackage()
	p.Name = "conflict"
	for i, typ := range []entity.AddrType{entity.AddrTypeData, entity.AddrTypeText} {
		p.AddSymbolCoverage(&entity.Addr{AddrPos: &entity.AddrPos{Addr: 100 + uint64(i)*5, Size: 20, Type: typ}, Pkg: p, SourceType: entity.AddrSourceDwarf})
	}
	return p
}

func TestPackageCoverageConflictDoesNotPanic(t *testing.T) {
	p := conflictingPackage()
	for range 2 {
		require.NotPanics(t, func() {
			var conflict *entity.ErrAddrCoverageConflict
			require.ErrorAs(t, p.AssignPackageSize(), &conflict)
			require.Zero(t, p.Size)
		})
	}
}
