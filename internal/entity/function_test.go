package entity_test

import (
	"slices"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestFunctionCodeRegionsPreserveGaps(t *testing.T) {
	regions := []entity.AddrPos{{Addr: 100, Size: 10, Type: entity.AddrTypeText}, {Addr: 200, Size: 20, Type: entity.AddrTypeText}}
	fn := &entity.Function{Addr: 100, CodeSize: 30, Ranges: regions}
	require.Equal(t, regions, slices.Collect(fn.CodeRegions))
	fn.Ranges = nil
	require.Equal(t, []entity.AddrPos{{Addr: 100, Size: 30, Type: entity.AddrTypeText}}, slices.Collect(fn.CodeRegions))
}
