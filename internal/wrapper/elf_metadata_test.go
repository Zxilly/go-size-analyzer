package wrapper

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func elfMetadataFixture(class elf.Class, order binary.ByteOrder) (*ElfWrapper, []byte) {
	data := make([]byte, 1024)
	stride := uint64(24)
	if class == elf.ELFCLASS32 {
		stride = 16
	}
	names := []byte("\x00pkg.Shared\x00")
	copy(data[0x200:], names)
	sections := []*elf.Section{
		{SectionHeader: elf.SectionHeader{Type: elf.SHT_NULL}},
		{SectionHeader: elf.SectionHeader{Name: ".text", Type: elf.SHT_PROGBITS, Flags: elf.SHF_ALLOC | elf.SHF_EXECINSTR, Addr: 0x1000, Offset: 0x300, Size: 32, FileSize: 32}},
		{SectionHeader: elf.SectionHeader{Name: ".data", Type: elf.SHT_PROGBITS, Flags: elf.SHF_ALLOC | elf.SHF_WRITE, Addr: 0x2000, Offset: 0x320, Size: 32, FileSize: 32}},
		{SectionHeader: elf.SectionHeader{Name: ".symtab", Type: elf.SHT_SYMTAB, Offset: 0x40, Size: 7 * stride, FileSize: 7 * stride, Entsize: stride, Link: 4}},
		{SectionHeader: elf.SectionHeader{Name: ".strtab", Type: elf.SHT_STRTAB, Offset: 0x200, Size: uint64(len(names)), FileSize: uint64(len(names))}},
		{SectionHeader: elf.SectionHeader{Name: ".bss", Type: elf.SHT_NOBITS, Flags: elf.SHF_ALLOC | elf.SHF_WRITE, Addr: 0x3000, Size: 32}},
		{SectionHeader: elf.SectionHeader{Name: ".go.func", Type: elf.SHT_PROGBITS, Flags: elf.SHF_ALLOC, Addr: 0x4000, Offset: 0x340, Size: 16, FileSize: 16}},
	}
	for i, entry := range []struct {
		section uint16
		value   uint64
		typ     elf.SymType
	}{{1, 0x1000, elf.STT_FUNC}, {2, 0x2000, elf.STT_OBJECT}, {0, 0, elf.STT_FUNC}, {1, 0x1000, elf.STT_SECTION}, {5, 0x3000, elf.STT_OBJECT}, {1, 0x1020, elf.STT_FUNC}} {
		b := data[0x40+uint64(i+1)*stride:]
		order.PutUint32(b, 1)
		if stride == 16 {
			order.PutUint32(b[4:], uint32(entry.value))
			b[12] = byte(entry.typ)
			order.PutUint16(b[14:], entry.section)
		} else {
			b[4] = byte(entry.typ)
			order.PutUint16(b[6:], entry.section)
			order.PutUint64(b[8:], entry.value)
		}
	}
	return &ElfWrapper{file: &elf.File{FileHeader: elf.FileHeader{Class: class, ByteOrder: order}, Sections: sections}}, data
}

func TestELFFileMetadataSymbolRecords(t *testing.T) {
	for _, class := range []elf.Class{elf.ELFCLASS32, elf.ELFCLASS64} {
		for name, order := range metadataTestOrders() {
			t.Run(class.String()+"/"+name, func(t *testing.T) {
				w, data := elfMetadataFixture(class, order)
				got, err := LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
				require.NoError(t, err)
				require.Len(t, got.Symbols, 2)
				require.Equal(t, entity.AddrTypeText, got.Symbols[0].Type)
				require.Equal(t, entity.AddrTypeData, got.Symbols[1].Type)
				require.Equal(t, got.Symbols[0].Ranges[1], got.Symbols[1].Ranges[1], "shared strings retain the same physical range")
				require.Equal(t, uint64(len("pkg.Shared")+1), got.Symbols[0].Ranges[1].Size)
				requireMetadataRanges(t, got, data)
				found := false
				for _, r := range got.Ranges {
					found = found || r.Offset == 0x340 && r.Size == 16
				}
				require.True(t, found, "Go metadata section is recognized")
			})
		}
	}
}

func TestELFFileMetadataExtendedSectionIndices(t *testing.T) {
	w, data := elfMetadataFixture(elf.ELFCLASS64, binary.LittleEndian)
	binary.LittleEndian.PutUint16(data[0x40+24+6:], uint16(elf.SHN_XINDEX))
	w.file.Sections = append(w.file.Sections, &elf.Section{SectionHeader: elf.SectionHeader{Type: elf.SHT_SYMTAB_SHNDX, Offset: 0x380, Size: 28, FileSize: 28, Link: 3}})
	binary.LittleEndian.PutUint32(data[0x384:], 1)
	got, err := LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
	require.NoError(t, err)
	require.Len(t, got.Symbols, 2)
	w.file.Sections = w.file.Sections[:len(w.file.Sections)-1]
	got, err = LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
	require.Error(t, err)
	require.Empty(t, got)
}

func TestELFFileMetadataRejectsMalformedTables(t *testing.T) {
	for name, mutate := range map[string]func(*ElfWrapper, []byte){
		"overflow":            func(w *ElfWrapper, _ []byte) { w.file.Sections[3].Offset = math.MaxUint64 },
		"stride":              func(w *ElfWrapper, _ []byte) { w.file.Sections[3].Entsize = 1 },
		"partial record":      func(w *ElfWrapper, _ []byte) { w.file.Sections[3].FileSize-- },
		"bad string link":     func(w *ElfWrapper, _ []byte) { w.file.Sections[3].Link = 100 },
		"bad string offset":   func(_ *ElfWrapper, data []byte) { binary.LittleEndian.PutUint32(data[0x40+24:], 100) },
		"unterminated string": func(_ *ElfWrapper, data []byte) { data[0x200+len("\x00pkg.Shared")] = 'x' },
		"target outside file": func(w *ElfWrapper, _ []byte) { w.file.Sections[1].FileSize = math.MaxUint64 },
	} {
		t.Run(name, func(t *testing.T) {
			w, data := elfMetadataFixture(elf.ELFCLASS64, binary.LittleEndian)
			mutate(w, data)
			got, err := LoadFileMetadata(w, bytes.NewReader(data), uint64(len(data)))
			require.Error(t, err)
			require.Empty(t, got)
		})
	}
	w, data := elfMetadataFixture(elf.ELFCLASS64, binary.LittleEndian)
	got, err := LoadFileMetadata(w, bytes.NewReader(data[:100]), uint64(len(data)))
	require.Error(t, err)
	require.Empty(t, got)
}
