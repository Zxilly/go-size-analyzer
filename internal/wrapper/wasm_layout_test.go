package wrapper

import (
	"bytes"
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
