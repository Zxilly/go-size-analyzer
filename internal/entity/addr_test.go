package entity_test

import (
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestAddrPosString(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 4096,
		Size: 256,
		Type: entity.AddrTypeData,
	}

	expected := "Addr: 1000 Size: 100 Type: data"
	result := addrPos.String()

	assert.Equal(t, expected, result)
}

func TestAddrPosStringWithDifferentType(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 4096,
		Size: 256,
		Type: entity.AddrTypeText,
	}

	expected := "Addr: 1000 Size: 100 Type: text"
	result := addrPos.String()

	assert.Equal(t, expected, result)
}

func TestAddrPosStringWithZeroSize(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 4096,
		Size: 0,
		Type: entity.AddrTypeData,
	}

	expected := "Addr: 1000 Size: 0 Type: data"
	result := addrPos.String()

	assert.Equal(t, expected, result)
}

func TestAddrPosStringWithZeroAddr(t *testing.T) {
	addrPos := &entity.AddrPos{
		Addr: 0,
		Size: 256,
		Type: entity.AddrTypeData,
	}

	expected := "Addr: 0 Size: 100 Type: data"
	result := addrPos.String()

	assert.Equal(t, expected, result)
}

func TestMergeCoverage(t *testing.T) {
	cov1 := entity.AddrCoverage{
		&entity.CoveragePart{
			Pos:   &entity.AddrPos{Addr: 4096, Size: 256, Type: entity.AddrTypeData},
			Addrs: []*entity.Addr{{}},
		},
	}
	cov2 := entity.AddrCoverage{
		&entity.CoveragePart{
			Pos:   &entity.AddrPos{Addr: 4351, Size: 256, Type: entity.AddrTypeData},
			Addrs: []*entity.Addr{{}},
		},
	}

	expected := entity.AddrCoverage{
		&entity.CoveragePart{
			Pos:   &entity.AddrPos{Addr: 4096, Size: 511, Type: entity.AddrTypeData},
			Addrs: nil,
		},
	}

	result, err := entity.MergeAndCleanCoverage([]entity.AddrCoverage{cov1, cov2})
	assert.NoError(t, err)

	// reset result Addrs
	lo.ForEach(result, func(part *entity.CoveragePart, _ int) {
		part.Addrs = nil
	})

	assert.Equal(t, expected, result)

	cov3 := entity.AddrCoverage{
		&entity.CoveragePart{
			Pos:   &entity.AddrPos{Addr: 4096, Size: 256, Type: entity.AddrTypeText},
			Addrs: []*entity.Addr{{}},
		},
	}
	cov4 := entity.AddrCoverage{
		&entity.CoveragePart{
			Pos:   &entity.AddrPos{Addr: 4160, Size: 128, Type: entity.AddrTypeData}, // 与 cov3 有重叠
			Addrs: []*entity.Addr{{}},
		},
	}

	_, err = entity.MergeAndCleanCoverage([]entity.AddrCoverage{cov3, cov4})
	assert.Error(t, err)
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
	}

	coveragePart := &entity.CoveragePart{
		Pos:   addrPos,
		Addrs: []*entity.Addr{addr1, addr2},
	}

	expected := "Pos: Addr: 1000 Size: 100 Type: data\n" +
		"Addr: 0x1000 Size: 256 pkg: nil SourceType: disasm\n" +
		"Addr: 0x1000 Size: 256 pkg: nil SourceType: symbol"
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

	expected := "Pos: Addr: 1000 Size: 100 Type: data"
	result := coveragePart.String()

	assert.Equal(t, expected, result)
}
