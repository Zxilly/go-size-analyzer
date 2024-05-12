package entity_test

import (
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/assert"
	"testing"
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
