package entity_test

import (
	"math/rand/v2"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
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

	result, err := entity.MergeAndCleanCoverage([]entity.AddrCoverage{cov1, cov2})
	require.NoError(t, err)

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

func FuzzMergeAndCleanCoverage(f *testing.F) {
	f.Add(uint64(42), uint64(233))

	f.Fuzz(func(t *testing.T, seed1, seed2 uint64) {
		r := rand.New(rand.NewPCG(seed1, seed2)) //nolint:gosec

		partCnt := rand.IntN(1024) + 2 //nolint:gosec

		coves := make([]entity.AddrCoverage, 0, partCnt)

		// we assume an addr space from 1 to 524288 (512KB),
		// and each addr has a size from 1 to 65536 (64KB)

		const maxAddr = 524288
		const maxSize = 65536

		for range partCnt {
			addrPos := &entity.AddrPos{
				Addr: r.Uint64N(maxAddr) + 1,
				Size: r.Uint64N(maxSize),
				Type: entity.AddrTypeData,
			}

			cov := entity.CoveragePart{
				Pos: addrPos,
				Addrs: []*entity.Addr{
					{
						AddrPos:    addrPos,
						SourceType: entity.AddrSourceDwarf,
					},
				},
			}

			coves = append(coves, entity.AddrCoverage{&cov})
		}

		merged, err := entity.MergeAndCleanCoverage(coves)
		require.NoError(t, err)

		oldAddrSpace := make([]bool, maxAddr+maxSize+1)
		newAddrSpace := make([]bool, maxAddr+maxSize+1)

		for _, cov := range coves {
			for _, part := range cov {
				for i := part.Pos.Addr; i < part.Pos.Addr+part.Pos.Size; i++ {
					oldAddrSpace[i] = true
				}
			}
		}

		for _, part := range merged {
			for i := part.Pos.Addr; i < part.Pos.Addr+part.Pos.Size; i++ {
				newAddrSpace[i] = true
			}
		}

		for i := 1; i <= maxAddr; i++ {
			assert.Equal(t, oldAddrSpace[i], newAddrSpace[i])
		}
	})
}
