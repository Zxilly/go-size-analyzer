package dwarf

import (
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypedSliceCapacityAndMalformedHeaders(t *testing.T) {
	byteType := &dwarf.UintType{BasicType: dwarf.BasicType{CommonType: dwarf.CommonType{Name: "uint8", ByteSize: 1}}}
	intType := &dwarf.IntType{BasicType: dwarf.BasicType{CommonType: dwarf.CommonType{Name: "int", ByteSize: 8}}}
	ptrType := &dwarf.PtrType{CommonType: dwarf.CommonType{ByteSize: 8}, Type: byteType}
	slice := &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 24}, StructName: "[]uint8", Field: []*dwarf.StructField{
		{Name: "array", Type: ptrType, ByteOffset: 0},
		{Name: "len", Type: intType, ByteOffset: 8},
		{Name: "cap", Type: intType, ByteOffset: 16},
	}}
	for _, tc := range []struct {
		name             string
		length, capacity uint64
		short, accepted  bool
		wantError        string
	}{
		{"capacity exceeds length", 1, 3, false, true, ""},
		{"length exceeds capacity", 4, 3, false, false, "invalid slice length or capacity"},
		{"negative capacity", 1, ^uint64(0), false, false, "negative header length"},
		{"beyond backing bytes", 1, 5, false, false, ""},
		{"truncated header", 1, 3, true, false, "short memory read"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			header := make([]byte, 24)
			binary.LittleEndian.PutUint64(header, 8192)
			binary.LittleEndian.PutUint64(header[8:], tc.length)
			binary.LittleEndian.PutUint64(header[16:], tc.capacity)
			memory := MemoryReader{
				ByteOrder: binary.LittleEndian, Limit: 64,
				Read: func(addr, size uint64) ([]byte, error) {
					if addr != 4096 || size != 24 {
						return nil, errors.New("unexpected read")
					}
					if tc.short {
						return header[:20], nil
					}
					return header, nil
				},
				Valid: func(addr, size uint64) bool { return addr == 8192 && size <= 4 },
			}
			w := typedDataWalker{memory: memory, references: map[dwarf.Type]bool{}}
			require.NotPanics(t, func() { w.walk(slice, 4096, "", 0) })
			if tc.wantError != "" {
				require.ErrorContains(t, w.err, tc.wantError)
			} else {
				require.NoError(t, w.err)
			}
			if tc.accepted {
				require.Equal(t, []Content{{Name: "[]uint8", Addr: 8192, Size: 3}}, w.contents)
			} else {
				require.Empty(t, w.contents)
			}
		})
	}
}

func TestTypedAggregateMalformedFieldsReturnError(t *testing.T) {
	for _, name := range []string{"ordinary struct", "string", "[]uint8", "embed.FS"} {
		for _, field := range []*dwarf.StructField{nil, {Name: "missing"}} {
			t.Run(fmt.Sprintf("%s/%v", name, field), func(t *testing.T) {
				typ := &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 8}, StructName: name, Field: []*dwarf.StructField{field}}
				reads := 0
				w := typedDataWalker{references: map[dwarf.Type]bool{}, memory: MemoryReader{
					ByteOrder: binary.LittleEndian,
					Read: func(uint64, uint64) ([]byte, error) {
						reads++
						return nil, errors.New("unexpected read")
					},
				}}
				require.NotPanics(t, func() { w.walk(typ, 4096, "", 0) })
				require.ErrorContains(t, w.err, "has no type")
				require.Empty(t, w.contents)
				require.Zero(t, reads)
			})
		}
	}
}

func TestTypedAggregatePropagatesNestedHeaderErrors(t *testing.T) {
	malformed := stringLayout(3, 6, 3)
	aggregate := &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: 6}, Field: []*dwarf.StructField{{Name: "Text", Type: malformed}}}
	array := &dwarf.ArrayType{Type: aggregate, Count: 10000}
	reads := 0
	w := typedDataWalker{references: map[dwarf.Type]bool{}, memory: MemoryReader{
		ByteOrder: binary.LittleEndian,
		Read: func(uint64, uint64) ([]byte, error) {
			reads++
			return nil, errors.New("unexpected read")
		},
	}}
	require.NotPanics(t, func() { w.walk(array, 4096, "", 0) })
	require.ErrorContains(t, w.err, "unsupported width")
	require.Empty(t, w.contents)
	require.Equal(t, 3, w.visited, "stop after the first malformed array element")
	require.Zero(t, reads)
}

func TestTypedStringUsesValidatedHeaderReader(t *testing.T) {
	for _, tc := range []struct {
		name      string
		reader    MemoryReader
		wantError string
	}{
		{"missing reader", MemoryReader{ByteOrder: binary.LittleEndian}, "incomplete memory reader"},
		{"missing byte order", MemoryReader{Read: func(uint64, uint64) ([]byte, error) { return nil, nil }}, "incomplete memory reader"},
		{"file limit", MemoryReader{ByteOrder: binary.LittleEndian, Limit: 4, Read: func(uint64, uint64) ([]byte, error) { return nil, nil }}, "exceeds file size"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := typedDataWalker{references: map[dwarf.Type]bool{}, memory: tc.reader}
			require.NotPanics(t, func() { w.walk(stringLayout(8, 16, 8), 4096, "", 0) })
			require.ErrorContains(t, w.err, tc.wantError)
			require.Empty(t, w.contents)
		})
	}
}
