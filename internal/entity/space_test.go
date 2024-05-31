package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func TestAddrSpaceGetReturnsExistingAddr(t *testing.T) {
	a := entity.AddrSpace{1: &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1}}}
	addr, ok := a.Get(1)
	assert.True(t, ok)
	assert.Equal(t, uint64(1), addr.Addr)
}

func TestAddrSpaceGetReturnsNilForNonExistingAddr(t *testing.T) {
	a := entity.AddrSpace{1: &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1}}}
	addr, ok := a.Get(2)
	assert.False(t, ok)
	assert.Nil(t, addr)
}

func TestAddrSpaceInsertAddsNewAddr(t *testing.T) {
	a := entity.AddrSpace{}
	addr := &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1}}
	a.Insert(addr)
	assert.Equal(t, addr, a[1])
}

func TestAddrSpaceInsertUpdatesExistingAddrWithLargerSize(t *testing.T) {
	a := entity.AddrSpace{1: &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1, Size: 1}}}
	addr := &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1, Size: 2}}
	a.Insert(addr)
	assert.Equal(t, addr, a[1])
}

func TestAddrSpaceInsertIgnoresExistingAddrWithSmallerSize(t *testing.T) {
	existingAddr := &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1, Size: 2}}
	a := entity.AddrSpace{1: existingAddr}
	addr := &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1, Size: 1}}
	a.Insert(addr)
	assert.Equal(t, existingAddr, a[1])
}

func TestMergeAddrSpaceMergesMultipleAddrSpaces(t *testing.T) {
	a1 := entity.AddrSpace{1: &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1, Size: 1}}}
	a2 := entity.AddrSpace{2: &entity.Addr{AddrPos: &entity.AddrPos{Addr: 2, Size: 2}}}
	merged := entity.MergeAddrSpace(a1, a2)

	assert.Len(t, merged, 2)
	assert.Equal(t, a1[1], merged[1])
	assert.Equal(t, a2[2], merged[2])
}

func TestMergeAddrSpacePrefersLargerSize(t *testing.T) {
	a1 := entity.AddrSpace{1: &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1, Size: 1}}}
	a2 := entity.AddrSpace{1: &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1, Size: 2}}}
	merged := entity.MergeAddrSpace(a1, a2)

	assert.Len(t, merged, 1)
	assert.Equal(t, a2[1], merged[1])
}

func TestToDirtyCoverageReturnsCoverageParts(t *testing.T) {
	a := entity.AddrSpace{1: &entity.Addr{AddrPos: &entity.AddrPos{Addr: 1, Size: 1}}}
	coverage := a.ToDirtyCoverage()

	assert.Len(t, coverage, 1)
	assert.Equal(t, a[1].AddrPos, coverage[0].Pos)
	assert.Equal(t, a[1], coverage[0].Addrs[0])
}
