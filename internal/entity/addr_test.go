package entity_test

import (
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

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

	result, err := entity.MergeCoverage([]entity.AddrCoverage{cov1, cov2})
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

	_, err = entity.MergeCoverage([]entity.AddrCoverage{cov3, cov4})
	assert.Error(t, err)
}
