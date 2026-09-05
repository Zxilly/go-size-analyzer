package entity_test

import (
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestGlobalCoveragePreservesPackageSizes(t *testing.T) {
	add := func(p *entity.Package, addr, size uint64) *entity.Addr {
		symbol := entity.NewSymbol(p.Name, addr, size, entity.AddrTypeData)
		ap := &entity.Addr{AddrPos: &entity.AddrPos{Addr: addr, Size: size, Type: entity.AddrTypeData}, Pkg: p, Symbol: symbol, SourceType: entity.AddrSourceSymbol}
		p.AddSymbol(symbol, ap)
		return ap
	}
	a, b := entity.NewPackage(), entity.NewPackage()
	a.Name = "a"
	b.Name = "b"
	original := add(a, 100, 10)
	add(b, 105, 20)
	ca, cb := a.GetPackageCoverage(), b.GetPackageCoverage()
	for range 2 {
		global, err := entity.MergeAndCleanCoverage([]entity.AddrCoverage{ca, cb})
		require.NoError(t, err)
		require.Equal(t, uint64(25), global[0].Pos.Size)
		a.AssignPackageSize()
		b.AssignPackageSize()
		require.Equal(t, uint64(10), a.Size)
		require.Equal(t, uint64(20), b.Size)
		require.Equal(t, uint64(10), original.Size)
		require.Len(t, ca[0].Addrs, 1)
	}
}
