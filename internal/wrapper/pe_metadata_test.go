package wrapper

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func peMetadataFixture() (*PeWrapper, []byte) {
	data := make([]byte, 1024)
	const count, off = 6, 0x40
	strOff := off + count*18
	names := []byte("\x00\x00\x00\x00pkg.variable\x00")
	binary.LittleEndian.PutUint32(names, uint32(len(names)))
	copy(data[strOff:], names)
	set := func(i int, name string, section uint16, value uint32, class, aux byte) {
		b := data[off+i*18:]
		if name == "" {
			binary.LittleEndian.PutUint32(b[4:], 4)
		} else {
			copy(b[:8], name)
		}
		binary.LittleEndian.PutUint32(b[8:], value)
		binary.LittleEndian.PutUint16(b[12:], section)
		b[16], b[17] = class, aux
	}
	set(0, "pkg.Fn", 1, 0, 2, 1)
	set(1, "aux", 1, 0, 2, 255) // auxiliary bytes must never become a symbol
	set(2, "", 2, 0, 2, 0)
	set(3, "", 2, 8, 2, 0)
	set(4, ".file", 0xfffe, 0, 103, 0)
	set(5, ".text", 1, 0, 3, 0)
	f := &pe.File{FileHeader: pe.FileHeader{PointerToSymbolTable: off, NumberOfSymbols: count}, Sections: []*pe.Section{
		{SectionHeader: pe.SectionHeader{Name: ".text", VirtualAddress: 0x1000, VirtualSize: 32, Size: 32, Offset: 0x300, Characteristics: pe.IMAGE_SCN_CNT_CODE}},
		{SectionHeader: pe.SectionHeader{Name: ".data", VirtualAddress: 0x2000, VirtualSize: 32, Size: 32, Offset: 0x320, Characteristics: pe.IMAGE_SCN_CNT_INITIALIZED_DATA}},
	}}
	return &PeWrapper{file: f, imageBase: 0x400000}, data
}

func TestPEFileMetadataSymbolRecords(t *testing.T) {
	w, data := peMetadataFixture()
	got, err := LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
	require.NoError(t, err)
	require.Len(t, got.Symbols, 3)
	require.Equal(t, "pkg.Fn", got.Symbols[0].Name)
	require.Equal(t, uint64(0x401000), got.Symbols[0].Addr)
	require.Equal(t, entity.AddrTypeText, got.Symbols[0].Type)
	require.Equal(t, uint64(36), got.Symbols[0].Ranges[0].Size, "auxiliary record belongs to the preceding symbol")
	require.Len(t, got.Symbols[0].Ranges, 1, "short names are already in the symbol record")
	require.Equal(t, entity.AddrTypeData, got.Symbols[1].Type)
	require.Equal(t, got.Symbols[1].Ranges[1], got.Symbols[2].Ranges[1])
	require.Equal(t, uint64(len("pkg.variable")+1), got.Symbols[1].Ranges[1].Size)
	requireMetadataRanges(t, got, data)
}

func TestPEFileMetadataRejectsMalformedTables(t *testing.T) {
	for name, mutate := range map[string]func(*PeWrapper, []byte){
		"outside table":           func(w *PeWrapper, _ []byte) { w.file.PointerToSymbolTable = math.MaxUint32 },
		"huge count":              func(w *PeWrapper, _ []byte) { w.file.NumberOfSymbols = math.MaxUint32 },
		"missing offset":          func(w *PeWrapper, _ []byte) { w.file.PointerToSymbolTable = 0 },
		"auxiliary overflow":      func(_ *PeWrapper, data []byte) { data[0x40+17] = 6 },
		"string size":             func(_ *PeWrapper, data []byte) { binary.LittleEndian.PutUint32(data[0x40+6*18:], 3) },
		"huge string table":       func(_ *PeWrapper, data []byte) { binary.LittleEndian.PutUint32(data[0x40+6*18:], math.MaxUint32) },
		"string header reference": func(_ *PeWrapper, data []byte) { binary.LittleEndian.PutUint32(data[0x40+2*18+4:], 2) },
		"unterminated name":       func(_ *PeWrapper, data []byte) { data[0x40+6*18+4+len("pkg.variable")] = 'x' },
		"address overflow":        func(w *PeWrapper, _ []byte) { w.imageBase = math.MaxUint64 },
	} {
		t.Run(name, func(t *testing.T) {
			w, data := peMetadataFixture()
			mutate(w, data)
			got, err := LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
			require.Error(t, err)
			require.Empty(t, got)
		})
	}
}

func TestPEFileMetadataExcludesMemoryOnlyTargets(t *testing.T) {
	w, data := peMetadataFixture()
	w.file.Sections[1].Characteristics = pe.IMAGE_SCN_CNT_UNINITIALIZED_DATA
	got, err := LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
	require.NoError(t, err)
	require.Len(t, got.Symbols, 1)
	require.NotEmpty(t, got.Ranges)
	w.file.Sections[1].Characteristics = pe.IMAGE_SCN_CNT_INITIALIZED_DATA
	w.file.Sections[1].Name = ".debug_data"
	got, err = LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
	require.NoError(t, err)
	require.Len(t, got.Symbols, 1, "debug data is not a runtime data object")
}
