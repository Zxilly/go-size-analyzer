package wrapper

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/types"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func machoMetadataFixture(is64 bool, order binary.ByteOrder) (*MachoWrapper, []byte, int) {
	data := make([]byte, 0x500)
	header, segment, stride, magic := 28, 56, 12, uint32(0xfeedface)
	if is64 {
		header, segment, stride, magic = 32, 72, 16, 0xfeedfacf
	}
	order.PutUint32(data, magic)
	order.PutUint32(data[16:], 3)
	order.PutUint32(data[20:], uint32(segment+24+16))
	cmd := data[header:]
	order.PutUint32(cmd, 1)
	order.PutUint32(cmd[4:], uint32(segment))
	copy(cmd[8:24], "__LINKEDIT")
	if is64 {
		order.PutUint32(cmd, 0x19)
		order.PutUint64(cmd[40:], 0x300)
		order.PutUint64(cmd[48:], 0x100)
	} else {
		order.PutUint32(cmd[32:], 0x300)
		order.PutUint32(cmd[36:], 0x100)
	}
	symCommand := header + segment
	cmd = data[symCommand:]
	order.PutUint32(cmd, 2)
	order.PutUint32(cmd[4:], 24)
	order.PutUint32(cmd[8:], 0x300)
	order.PutUint32(cmd[12:], 6)
	order.PutUint32(cmd[16:], 0x380)
	names := []byte("\x00_pkg.Fn\x00")
	order.PutUint32(cmd[20:], uint32(len(names)))
	copy(data[0x380:], names)
	cmd = data[symCommand+24:]
	order.PutUint32(cmd, 0x26) // function starts payload outside any section
	order.PutUint32(cmd[4:], 16)
	order.PutUint32(cmd[8:], 0x3c0)
	order.PutUint32(cmd[12:], 12)
	for i, entry := range []struct {
		typ     byte
		section byte
		value   uint64
	}{{0xf, 1, 0x1000}, {0xf, 2, 0x2000}, {1, 0, 0}, {0xef, 1, 0x1000}, {0xf, 3, 0x3000}, {0xf, 1, 0x1020}} {
		b := data[0x300+i*stride:]
		order.PutUint32(b, 1)
		b[4], b[5] = entry.typ, entry.section
		if is64 {
			order.PutUint64(b[8:], entry.value)
		} else {
			order.PutUint32(b[8:], uint32(entry.value))
		}
	}
	f := &macho.File{FileTOC: macho.FileTOC{Sections: []*types.Section{
		{SectionHeader: types.SectionHeader{Name: "__text", Seg: "__TEXT", Addr: 0x1000, Offset: 0x400, Size: 32}},
		{SectionHeader: types.SectionHeader{Name: "__data", Seg: "__DATA", Addr: 0x2000, Offset: 0x420, Size: 32}},
		{SectionHeader: types.SectionHeader{Name: "__bss", Seg: "__DATA", Addr: 0x3000, Size: 32, Flags: 1}},
		{SectionHeader: types.SectionHeader{Name: "__go_func", Seg: "__DATA_CONST", Addr: 0x4000, Offset: 0x440, Size: 16}},
	}}}
	return &MachoWrapper{file: f}, data, symCommand
}

func TestMachoFileMetadataSymbolRecords(t *testing.T) {
	for _, is64 := range []bool{false, true} {
		for name, order := range metadataTestOrders() {
			t.Run(name+map[bool]string{false: "32", true: "64"}[is64], func(t *testing.T) {
				w, data, _ := machoMetadataFixture(is64, order)
				got, err := LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
				require.NoError(t, err)
				require.Len(t, got.Symbols, 2)
				require.Equal(t, "pkg.Fn", got.Symbols[0].Name)
				require.Equal(t, entity.AddrTypeText, got.Symbols[0].Type)
				require.Equal(t, entity.AddrTypeData, got.Symbols[1].Type)
				require.Equal(t, got.Symbols[0].Ranges[1], got.Symbols[1].Ranges[1])
				require.Equal(t, uint64(len("_pkg.Fn")+1), got.Symbols[0].Ranges[1].Size, "range includes Go name decoration and NUL")
				requireMetadataRanges(t, got, data)
				found := make(map[string]bool)
				for _, r := range got.Ranges {
					found[r.Source] = true
				}
				require.False(t, found["macho-linkedit"], "a segment is not evidence for its unused gaps")
				require.True(t, found["macho-linkedit-data"])
				require.True(t, found["macho-metadata:__go_func"])
				for _, r := range got.Ranges {
					require.False(t, r.Offset <= 0x3f0 && 0x3f0-r.Offset < r.Size, "unreferenced LINKEDIT bytes remain unclassified")
				}
			})
		}
	}
}

func TestMachoFileMetadataRejectsMalformedTables(t *testing.T) {
	for name, mutate := range map[string]func([]byte, int){
		"command area":        func(data []byte, _ int) { binary.LittleEndian.PutUint32(data[20:], math.MaxUint32) },
		"command count":       func(data []byte, _ int) { binary.LittleEndian.PutUint32(data[16:], math.MaxUint32) },
		"short command":       func(data []byte, _ int) { binary.LittleEndian.PutUint32(data[36:], 4) },
		"oversized command":   func(data []byte, _ int) { binary.LittleEndian.PutUint32(data[36:], math.MaxUint32) },
		"segment overflow":    func(data []byte, _ int) { binary.LittleEndian.PutUint64(data[32+40:], math.MaxUint64) },
		"symbol count":        func(data []byte, sym int) { binary.LittleEndian.PutUint32(data[sym+12:], math.MaxUint32) },
		"string range":        func(data []byte, sym int) { binary.LittleEndian.PutUint32(data[sym+20:], math.MaxUint32) },
		"bad name":            func(data []byte, _ int) { binary.LittleEndian.PutUint32(data[0x300:], 100) },
		"unterminated name":   func(data []byte, _ int) { data[0x380+len("\x00_pkg.Fn")] = 'x' },
		"linkedit data range": func(data []byte, sym int) { binary.LittleEndian.PutUint32(data[sym+24+12:], math.MaxUint32) },
	} {
		t.Run(name, func(t *testing.T) {
			w, data, sym := machoMetadataFixture(true, binary.LittleEndian)
			mutate(data, sym)
			got, err := LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
			require.Error(t, err)
			require.Empty(t, got)
		})
	}
}

func TestMachoFileMetadataDynamicTables(t *testing.T) {
	for _, is64 := range []bool{false, true} {
		for name, order := range metadataTestOrders() {
			t.Run(name+map[bool]string{false: "32", true: "64"}[is64], func(t *testing.T) {
				w, data, sym := machoMetadataFixture(is64, order)
				pos := sym + 24 + 16
				dynamic := data[pos : pos+80]
				order.PutUint32(dynamic, 0xb)
				order.PutUint32(dynamic[4:], 80)
				for off := 32; off < 80; off += 8 {
					order.PutUint32(dynamic[off:], 0x480)
					order.PutUint32(dynamic[off+4:], 1)
				}
				dyld := data[pos+80 : pos+128]
				order.PutUint32(dyld, 0x80000022)
				order.PutUint32(dyld[4:], 48)
				for off := 8; off < 48; off += 8 {
					order.PutUint32(dyld[off:], 0x4c0)
					order.PutUint32(dyld[off+4:], 2)
				}
				order.PutUint32(data[16:], 5)
				order.PutUint32(data[20:], order.Uint32(data[20:])+128)
				got, err := LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
				require.NoError(t, err)
				var dynamicSizes, dyldSizes []uint64
				for _, r := range got.Ranges {
					switch r.Source {
					case "macho-dynamic-symbol-table":
						dynamicSizes = append(dynamicSizes, r.Size)
					case "macho-dyld-info":
						dyldSizes = append(dyldSizes, r.Size)
					default:
					}
				}
				moduleSize := uint64(52)
				if is64 {
					moduleSize = 56
				}
				require.Equal(t, []uint64{8, moduleSize, 4, 4, 8, 8}, dynamicSizes)
				require.Equal(t, []uint64{2, 2, 2, 2, 2}, dyldSizes)
				order.PutUint32(dynamic[36:], math.MaxUint32)
				got, err = LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
				require.Error(t, err)
				require.Empty(t, got)
			})
		}
	}
}
