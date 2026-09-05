package wrapper

import (
	"bytes"
	"os"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestWasmRawMappingsPreserveDuplicateCustomSections(t *testing.T) {
	data := []byte{0, 97, 115, 109, 1, 0, 0, 0, 0, 2, 1, 'x', 0, 2, 1, 'x', 11, 9, 1, 0, 0x41, 0x20, 0x0b, 3, 'a', 'b', 'c'}
	w := &WasmWrapper{}
	require.NoError(t, w.LoadRaw(bytes.NewReader(data), uint64(len(data))))
	require.Len(t, w.GetSections(0, 0), 3)
	require.Contains(t, w.sections, "x")
	require.Contains(t, w.sections, "x#2")
	require.Equal(t, []entity.FileMapping{{Addr: 32, FileRange: entity.FileRange{Offset: 24, Size: 3}}}, w.FileAddressMappings())
}

func TestWasmMappingsMatchMemoryInitializationOrder(t *testing.T) {
	w := &WasmWrapper{}
	expected := make([]uint64, 40)
	for i, span := range [][2]uint64{{20, 10}, {4, 3}, {10, 10}, {2, 5}, {6, 18}, {0, 1}, {25, 8}} {
		m := entity.FileMapping{Addr: span[0], FileRange: entity.FileRange{Offset: uint64(i+1) * 100, Size: span[1]}}
		w.addFileMapping(m)
		for j := uint64(0); j < m.Size; j++ {
			expected[m.Addr+j] = m.Offset + j
		}
	}
	actual := make([]uint64, len(expected))
	var previousEnd uint64
	for _, m := range w.FileAddressMappings() {
		require.GreaterOrEqual(t, m.Addr, previousEnd)
		previousEnd = m.Addr + m.Size
		for j := uint64(0); j < m.Size; j++ {
			actual[m.Addr+j] = m.Offset + j
		}
	}
	require.Equal(t, expected, actual)
}

func BenchmarkWasmRawLayout(b *testing.B) {
	data, err := os.ReadFile("../../scripts/bins/bin-js-1.27-wasm")
	if err != nil {
		b.Skipf("benchmark fixture unavailable: %v", err)
	}
	b.ReportAllocs()
	for b.Loop() {
		w := &WasmWrapper{}
		if err := w.LoadRaw(bytes.NewReader(data), uint64(len(data))); err != nil {
			b.Fatal(err)
		}
	}
}

func TestWasmLaterDataSegmentsReplaceAddressMappings(t *testing.T) {
	w := &WasmWrapper{}
	w.addFileMapping(entity.FileMapping{Addr: 100, FileRange: entity.FileRange{Offset: 10, Size: 20}})
	w.addFileMapping(entity.FileMapping{Addr: 105, FileRange: entity.FileRange{Offset: 50, Size: 5}})
	require.Equal(t, []entity.FileMapping{
		{Addr: 100, FileRange: entity.FileRange{Offset: 10, Size: 5}},
		{Addr: 105, FileRange: entity.FileRange{Offset: 50, Size: 5}},
		{Addr: 110, FileRange: entity.FileRange{Offset: 20, Size: 10}},
	}, w.FileAddressMappings())
}
