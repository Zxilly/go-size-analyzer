package wrapper

import (
	"bytes"
	"compress/zlib"
	"debug/elf"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/types"
	"github.com/stretchr/testify/require"
)

func TestNativeDuplicateSectionNames(t *testing.T) {
	names := []string{"data", "data#2", "data", "data"}
	ef := &elf.File{}
	pf := &pe.File{}
	mf := &macho.File{}
	for i, name := range names {
		ef.Sections = append(ef.Sections, &elf.Section{SectionHeader: elf.SectionHeader{Name: name, Type: elf.SHT_PROGBITS, Size: 1, FileSize: 1, Offset: uint64(i + 1)}})
		pf.Sections = append(pf.Sections, &pe.Section{SectionHeader: pe.SectionHeader{Name: name, Size: 1, Offset: uint32(i + 1)}})
		mf.Sections = append(mf.Sections, &types.Section{SectionHeader: types.SectionHeader{Name: name, Seg: "__DATA", Size: 1, Offset: uint32(i + 1)}})
	}
	for name, w := range map[string]RawFileWrapper{"ELF": &ElfWrapper{file: ef}, "PE": &PeWrapper{file: pf}, "Mach-O": NewMachoWrapper(mf)} {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				store := w.LoadSections()
				require.Len(t, store.Sections, len(names))
				for key, section := range store.Sections {
					require.Equal(t, key, section.Name)
				}
			})
		})
	}
}

func appendULEB(b []byte, n uint32) []byte {
	for n >= 128 {
		b = append(b, byte(n)|128)
		n >>= 7
	}
	return append(b, byte(n))
}

func adversarialWasm(count int, custom bool, order string) []byte {
	b := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	if custom {
		for range count {
			b = append(b, 0, 2, 1, 'x')
		}
		return b
	}
	payload := appendULEB(nil, uint32(count))
	for i := range count {
		addr := i * 2
		if order == "descending" {
			addr = (count - i) * 2
		}
		if order == "overlap" {
			addr = count - i
		}
		payload = append(payload, 0, 0x41)
		// Positive signed LEB128 needs an extra zero group when bit 6 is set.
		v := uint32(addr)
		for v >= 64 {
			payload = append(payload, byte(v)|128)
			v >>= 7
		}
		payload = append(payload, byte(v), 0x0b, 2, 'a', 'b')
	}
	b = append(b, 11)
	b = appendULEB(b, uint32(len(payload)))
	return append(b, payload...)
}

func BenchmarkHostileWasmLayout(b *testing.B) {
	for _, kind := range []string{"ascending", "descending", "overlap", "custom"} {
		for _, n := range []int{1000, 4000, 16000} {
			b.Run(fmt.Sprintf("%s/%d", kind, n), func(b *testing.B) {
				data := adversarialWasm(n, kind == "custom", kind)
				b.ReportAllocs()
				for b.Loop() {
					w := &WasmWrapper{}
					if err := w.LoadRaw(bytes.NewReader(data), uint64(len(data))); err != nil {
						b.Fatal(err)
					}
					w.FileAddressMappings()
				}
			})
		}
	}
}

// A minimal accepted Mach-O with one file-backed segment and section.
func machoFixture(t *testing.T, name string, content []byte, fileSize uint64) *MachoWrapper {
	t.Helper()
	f, err := macho.NewFile(bytes.NewReader(machoBytes(name, content, fileSize)))
	require.NoError(t, err)
	return NewMachoWrapper(f)
}

func machoBytes(name string, content []byte, fileSize uint64) []byte {
	b := make([]byte, 32+72+80+len(content))
	put32 := func(off int, value uint32) { binary.LittleEndian.PutUint32(b[off:], value) }
	put64 := func(off int, value uint64) { binary.LittleEndian.PutUint64(b[off:], value) }
	put32(0, 0xfeedfacf)
	put32(4, uint32(types.CPUAmd64))
	put32(8, 3)
	put32(12, 2)
	put32(16, 1)
	put32(20, 152)
	put32(32, 0x19)
	put32(36, 152)
	copy(b[40:], "__DATA")
	put64(56, 4096)
	put64(64, fileSize)
	put64(72, 184)
	put64(80, fileSize)
	put32(96, 1)
	copy(b[104:], name)
	copy(b[120:], "__DATA")
	put64(136, 4096)
	put64(144, uint64(len(content)))
	put32(152, 184)
	copy(b[184:], content)
	return b
}

func TestMachoReadAddrValidatesFullBacking(t *testing.T) {
	w := machoFixture(t, "__data", []byte{1, 2, 3, 4}, 4)
	got, err := w.ReadAddr(4097, 2)
	require.NoError(t, err)
	require.Equal(t, []byte{2, 3}, got)
	for _, size := range []uint64{4, math.MaxInt64, math.MaxUint64 - 4097} {
		require.NotPanics(t, func() {
			got, err = w.ReadAddr(4097, size)
			require.ErrorIs(t, err, ErrAddrNotFound)
			require.Nil(t, got)
		})
	}
	w = machoFixture(t, "__data", []byte{1, 2, 3, 4}, math.MaxInt64)
	got, err = w.ReadAddr(4096, 1024*1024)
	require.ErrorIs(t, err, ErrAddrNotFound)
	require.Nil(t, got)
}

func TestMachoReadAddrUsesFirstOverlappingSegment(t *testing.T) {
	w := machoFixture(t, "__data", []byte{1, 2, 3, 4}, 4)
	first := w.file.Segments()[0]
	first.Memsz, first.Filesz = 8, 1
	later := *first
	later.Filesz = 4
	w.file.Loads = append(w.file.Loads, &later)
	got, err := w.ReadAddr(4096, 2)
	require.ErrorIs(t, err, ErrAddrNotFound)
	require.Nil(t, got)
}

func TestMachoCompressedDWARFRejectsHugeDeclaration(t *testing.T) {
	b := make([]byte, 12)
	copy(b, "ZLIB")
	binary.BigEndian.PutUint64(b[4:], math.MaxUint64)
	w := machoFixture(t, "__zdebug_info", b, uint64(len(b)))
	require.NotPanics(t, func() {
		_, err := w.DWARF()
		require.Error(t, err)
	})
}

func TestMachoCompressedDWARFChecksStream(t *testing.T) {
	var compressed bytes.Buffer
	z := zlib.NewWriter(&compressed)
	_, err := z.Write([]byte("debug strings\x00"))
	require.NoError(t, err)
	require.NoError(t, z.Close())
	for _, tc := range []struct {
		name    string
		size    uint64
		corrupt bool
		valid   bool
	}{
		{"valid", 14, false, true},
		{"short", 15, false, false},
		{"long", 13, false, false},
		{"checksum", 14, true, false},
		{"limit", (1 << 30) + 1, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 12)
			copy(data, "ZLIB")
			binary.BigEndian.PutUint64(data[4:], tc.size)
			data = append(data, compressed.Bytes()...)
			if tc.corrupt {
				data[len(data)-1] ^= 1
			}
			w := machoFixture(t, "__zdebug_str", data, uint64(len(data)))
			got, err := readMachoDWARFSection(w.file.Sections[0])
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, []byte("debug strings\x00"), got)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestWasmMappingSweepMatchesInitialization(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range 50 {
		w := &WasmWrapper{}
		expected := make([]uint64, 128)
		for i := 0; i < 200; i++ {
			addr := rng.Uint64N(128)
			size := rng.Uint64N(129 - addr)
			m := entity.FileMapping{Addr: addr, FileRange: entity.FileRange{Offset: uint64(i+1) * 1000, Size: size}}
			w.addFileMapping(m)
			for j := uint64(0); j < size; j++ {
				expected[addr+j] = m.Offset + j
			}
			if i%37 == 0 {
				w.FileAddressMappings()
			}
		}
		actual := make([]uint64, 128)
		for _, m := range w.FileAddressMappings() {
			for j := uint64(0); j < m.Size; j++ {
				actual[m.Addr+j] = m.Offset + j
			}
		}
		require.Equal(t, expected, actual)
	}
}

func TestWasmCustomSectionSuffixCollisions(t *testing.T) {
	b := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	for _, name := range []string{"x", "x#2", "x", "x#3", "x", "x#2"} {
		b = append(b, 0, byte(len(name)+1), byte(len(name)))
		b = append(b, name...)
	}
	w := &WasmWrapper{}
	require.NoError(t, w.LoadRaw(bytes.NewReader(b), uint64(len(b))))
	for _, name := range []string{"x", "x#2", "x#3", "x#3#2", "x#4", "x#2#2"} {
		require.Contains(t, w.sections, name)
	}
}

func TestParsedNativeDuplicateSections(t *testing.T) {
	t.Run("ELF", func(t *testing.T) {
		b := make([]byte, 64+4*64+8+2)
		copy(b, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
		binary.LittleEndian.PutUint16(b[16:], 2)
		binary.LittleEndian.PutUint16(b[18:], uint16(elf.EM_X86_64))
		binary.LittleEndian.PutUint32(b[20:], 1)
		binary.LittleEndian.PutUint64(b[40:], 64)
		binary.LittleEndian.PutUint16(b[52:], 64)
		binary.LittleEndian.PutUint16(b[58:], 64)
		binary.LittleEndian.PutUint16(b[60:], 4)
		binary.LittleEndian.PutUint16(b[62:], 1)
		copy(b[320:], "\x00.data\x00\x00")
		for i := 1; i < 4; i++ {
			s := b[64+i*64:]
			kind := uint32(elf.SHT_PROGBITS)
			off := uint64(328 + i - 2)
			size := uint64(1)
			if i == 1 {
				kind = uint32(elf.SHT_STRTAB)
				off = 320
				size = 8
			} else {
				binary.LittleEndian.PutUint32(s, 1)
			}
			binary.LittleEndian.PutUint32(s[4:], kind)
			binary.LittleEndian.PutUint64(s[24:], off)
			binary.LittleEndian.PutUint64(s[32:], size)
		}
		f, err := elf.NewFile(bytes.NewReader(b))
		require.NoError(t, err)
		require.Len(t, (&ElfWrapper{file: f}).LoadSections().Sections, 3)
	})
	t.Run("PE", func(t *testing.T) {
		b := make([]byte, 102)
		binary.LittleEndian.PutUint16(b, uint16(pe.IMAGE_FILE_MACHINE_AMD64))
		binary.LittleEndian.PutUint16(b[2:], 2)
		for i := 0; i < 2; i++ {
			s := b[20+40*i:]
			copy(s, ".data")
			binary.LittleEndian.PutUint32(s[16:], 1)
			binary.LittleEndian.PutUint32(s[20:], uint32(100+i))
		}
		f, err := pe.NewFile(bytes.NewReader(b))
		require.NoError(t, err)
		require.Len(t, (&PeWrapper{file: f}).LoadSections().Sections, 2)
	})
	t.Run("Mach-O", func(t *testing.T) {
		old := machoBytes("__data", []byte{1, 2}, 2)
		b := append([]byte(nil), old[:184]...)
		b = append(b, old[104:184]...)
		b = append(b, old[184:]...)
		binary.LittleEndian.PutUint32(b[20:], 232)
		binary.LittleEndian.PutUint32(b[36:], 232)
		binary.LittleEndian.PutUint64(b[72:], 264)
		binary.LittleEndian.PutUint32(b[96:], 2)
		for i := 0; i < 2; i++ {
			binary.LittleEndian.PutUint32(b[152+80*i:], uint32(264+i))
			binary.LittleEndian.PutUint64(b[144+80*i:], 1)
			binary.LittleEndian.PutUint64(b[136+80*i:], uint64(4096+i))
		}
		f, err := macho.NewFile(bytes.NewReader(b))
		require.NoError(t, err)
		require.Len(t, NewMachoWrapper(f).LoadSections().Sections, 2)
	})
}
