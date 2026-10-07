package dwarf

import (
	"debug/dwarf"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func stringLayout(ptrSize, size, lenOffset int64) *dwarf.StructType {
	return &dwarf.StructType{CommonType: dwarf.CommonType{ByteSize: size}, StructName: "string", Field: []*dwarf.StructField{
		{Name: "str", Type: &dwarf.PtrType{CommonType: dwarf.CommonType{ByteSize: ptrSize}, Type: &dwarf.UintType{BasicType: dwarf.BasicType{CommonType: dwarf.CommonType{Name: "uint8", ByteSize: 1}}}}},
		{Name: "len", ByteOffset: lenOffset, Type: &dwarf.IntType{BasicType: dwarf.BasicType{CommonType: dwarf.CommonType{Name: "int", ByteSize: ptrSize}}}},
	}}
}

func TestMalformedStringLayoutReturnsError(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  *dwarf.StructType
		data []byte
	}{
		{"unsupported width", stringLayout(3, 6, 3), make([]byte, 6)},
		{"negative offset", stringLayout(8, 16, -1), make([]byte, 16)},
		{"field outside struct", stringLayout(8, 16, 12), make([]byte, 16)},
		{"short memory", stringLayout(8, 16, 8), make([]byte, 1)},
		{"negative size", stringLayout(8, -1, 8), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := MemoryReader{ByteOrder: binary.LittleEndian, Read: func(uint64, uint64) ([]byte, error) { return tc.data, nil }}
			require.NotPanics(t, func() { _, _, err := readString(tc.typ, 100, reader); require.Error(t, err) })
		})
	}
}

func TestStringLengthRejectsNegativeAndAllowsTrailingPadding(t *testing.T) {
	typ := stringLayout(4, 12, 4)
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data, 100)
	binary.LittleEndian.PutUint32(data[4:], 0xffffffff)
	reader := MemoryReader{ByteOrder: binary.LittleEndian, Read: func(uint64, uint64) ([]byte, error) { return data, nil }}
	require.NotPanics(t, func() { _, _, err := readString(typ, 0, reader); require.Error(t, err) })
	binary.LittleEndian.PutUint32(data[4:], 5)
	addr, size, err := readString(typ, 0, reader)
	require.NoError(t, err)
	require.Equal(t, uint64(100), addr)
	require.Equal(t, uint64(5), size)
}

func TestMalformedParsedStringTypeReturnsError(t *testing.T) {
	// This layout is accepted by debug/dwarf.Type, including its three-byte pointer.
	abbrev := []byte{
		1, 0x11, 1, 0, 0,
		2, 0x24, 0, 3, 8, 0x0b, 0x0b, 0x3e, 0x0b, 0, 0,
		3, 0x0f, 0, 0x49, 0x13, 0x0b, 0x0b, 0, 0,
		4, 0x13, 1, 3, 8, 0x0b, 0x0b, 0, 0,
		5, 0x0d, 0, 3, 8, 0x49, 0x13, 0x38, 0x0b, 0, 0,
		6, 0x34, 0, 3, 8, 0x49, 0x13, 0, 0, 0,
	}
	body := []byte{4, 0, 0, 0, 0, 0, 8, 1}
	uintOffset := uint32(len(body) + 4)
	body = append(body, 2, 'u', 'i', 'n', 't', '8', 0, 1, 7)
	intOffset := uint32(len(body) + 4)
	body = append(body, 2, 'i', 'n', 't', 0, 3, 5)
	ptrOffset := uint32(len(body) + 4)
	body = append(body, 3)
	body = binary.LittleEndian.AppendUint32(body, uintOffset)
	body = append(body, 3)
	structOffset := uint32(len(body) + 4)
	body = append(body, 4, 's', 't', 'r', 'i', 'n', 'g', 0, 6, 5, 's', 't', 'r', 0)
	body = binary.LittleEndian.AppendUint32(body, ptrOffset)
	body = append(body, 0, 5, 'l', 'e', 'n', 0)
	body = binary.LittleEndian.AppendUint32(body, intOffset)
	body = append(body, 3, 0, 6, 'v', 0)
	body = binary.LittleEndian.AppendUint32(body, structOffset)
	body = append(body, 0)
	info := binary.LittleEndian.AppendUint32(nil, uint32(len(body)))
	info = append(info, body...)
	d, err := dwarf.New(abbrev, nil, nil, info, nil, nil, nil, nil)
	require.NoError(t, err)
	typ, err := d.Type(dwarf.Offset(structOffset))
	require.NoError(t, err)
	require.Equal(t, int64(3), typ.(*dwarf.StructType).Field[0].Type.Size())
	entry := &dwarf.Entry{Tag: dwarf.TagVariable, Field: []dwarf.Field{{Attr: dwarf.AttrType, Val: dwarf.Offset(structOffset)}}}
	reads := 0
	reader := MemoryReader{ByteOrder: binary.LittleEndian, Read: func(uint64, uint64) ([]byte, error) { reads++; return make([]byte, 6), nil }}
	require.NotPanics(t, func() {
		_, _, err = SizeForDWARFVar(d, entry, 0, reader)
		require.ErrorContains(t, err, "unsupported width")
	})
	require.Zero(t, reads)
}
