package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func TestAddrPosString(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 4096,
		Size: 256,
		Type: entity.AddrTypeData,
	}

	expected := "Addr: 1000 CodeSize: 100 Type: data"
	result := addrPos.String()

	assert.Equal(t, expected, result)
}

func TestAddrPosStringWithDifferentType(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 4096,
		Size: 256,
		Type: entity.AddrTypeText,
	}

	expected := "Addr: 1000 CodeSize: 100 Type: text"
	result := addrPos.String()

	assert.Equal(t, expected, result)
}

func TestAddrPosStringWithZeroSize(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 4096,
		Size: 0,
		Type: entity.AddrTypeData,
	}

	expected := "Addr: 1000 CodeSize: 0 Type: data"
	result := addrPos.String()

	assert.Equal(t, expected, result)
}

func TestAddrPosStringWithZeroAddr(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 0,
		Size: 256,
		Type: entity.AddrTypeData,
	}

	expected := "Addr: 0 CodeSize: 100 Type: data"
	result := addrPos.String()

	assert.Equal(t, expected, result)
}

func TestCoveragePartStringWithMultipleAddrs(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 4096,
		Size: 256,
		Type: entity.AddrTypeData,
	}

	addr1 := &entity.Addr{
		AddrPos:    addrPos,
		SourceType: entity.AddrSourceDisasm,
	}

	addr2 := &entity.Addr{
		AddrPos:    addrPos,
		SourceType: entity.AddrSourceSymbol,
		Function:   &entity.Function{Name: "main"},
		Pkg:        &entity.Package{Name: "main"},
	}

	coveragePart := &entity.CoveragePart{
		Pos:   addrPos,
		Addrs: []*entity.Addr{addr1, addr2},
	}

	expected := "Pos: Addr: 1000 CodeSize: 100 Type: data\n" +
		"AddrPos: Addr: 1000 CodeSize: 100 Type: data SourceType: disasm\n" +
		"AddrPos: Addr: 1000 CodeSize: 100 Type: data Pkg: main Function: main SourceType: symbol"
	result := coveragePart.String()

	assert.Equal(t, expected, result)
}

func TestCoveragePartStringWithNoAddrs(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 4096,
		Size: 256,
		Type: entity.AddrTypeData,
	}

	coveragePart := &entity.CoveragePart{
		Pos:   addrPos,
		Addrs: []*entity.Addr{},
	}

	expected := "Pos: Addr: 1000 CodeSize: 100 Type: data"
	result := coveragePart.String()

	assert.Equal(t, expected, result)
}

func TestErrorReturnsExpectedErrorMessage(t *testing.T) {
	addr := uint64(4096)
	pos1 := &entity.CoveragePart{
		Pos:   &entity.AddrPos{Addr: 4096, Size: 256, Type: entity.AddrTypeData},
		Addrs: []*entity.Addr{{}},
	}
	pos2 := &entity.CoveragePart{
		Pos:   &entity.AddrPos{Addr: 4351, Size: 256, Type: entity.AddrTypeData},
		Addrs: []*entity.Addr{{}},
	}

	err := &entity.ErrAddrCoverageConflict{
		Addr: addr,
		Pos1: pos1,
		Pos2: pos2,
	}

	expected := "addr 1000 pos Pos: Addr: 1000 CodeSize: 100 Type: data\n" +
		"AddrPos: <nil> SourceType:  and Pos: Addr: 10ff CodeSize: 100 Type: data\n" +
		"AddrPos: <nil> SourceType:  conflict"
	assert.Equal(t, expected, err.Error())
}
