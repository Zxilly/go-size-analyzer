package wrapper

import (
	"bytes"
	"compress/zlib"
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/types"
	"github.com/stretchr/testify/require"
)

func TestMachoDWARFExternalLinkFixture(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "bins", "bin-darwin-1.27-amd64-pie-cgo")
	f, err := macho.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("external-link Mach-O fixture is not installed")
	}
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	d, err := NewMachoWrapper(f).DWARF()
	require.NoError(t, err)
	r := d.Reader()
	entries := 0
	for {
		e, err := r.Next()
		require.NoError(t, err)
		if e == nil {
			break
		}
		entries++
	}
	require.Positive(t, entries)
}

func TestMachoDWARFFileOnlySectionsAndTruncatedNames(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		name := "plain"
		if compressed {
			name = "compressed"
		}
		t.Run(name, func(t *testing.T) {
			// A DWARF 5 compilation unit uses indexed strings, a line string,
			// and indexed address/range tables. Several Mach-O section names
			// exceed the format's 16-byte name field.
			abbrev := []byte{
				1, 0x11, 1, 0x03, 0x08, 0x72, 0x17, 0x1b, 0x1f, 0x74, 0x17, 0x73, 0x17, 0, 0,
				2, 0x2e, 0, 0x03, 0x25, 0x55, 0x23, 0, 0, 0,
			}
			info := []byte{5, 0, 1, 8, 0, 0, 0, 0, 1}
			info = append(info, "unit\x00"...)
			info = binary.LittleEndian.AppendUint32(info, 8)  // str_offsets_base
			info = binary.LittleEndian.AppendUint32(info, 0)  // comp_dir line_strp
			info = binary.LittleEndian.AppendUint32(info, 12) // rnglists_base
			info = binary.LittleEndian.AppendUint32(info, 8)  // addr_base
			info = append(info, 2, 0, 0, 0)                   // subprogram: name/ranges index 0; end CU
			info = append(binary.LittleEndian.AppendUint32(nil, uint32(len(info))), info...)
			strOffsets := []byte{8, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0}
			addresses := []byte{12, 0, 0, 0, 5, 0, 8, 0}
			addresses = binary.LittleEndian.AppendUint64(addresses, 0x1234)
			ranges := []byte{16, 0, 0, 0, 5, 0, 8, 0, 1, 0, 0, 0, 4, 0, 0, 0, 3, 0, 16, 0}
			sections := []machoDwarfSection{
				{"abbrev", abbrev},
				{"info", info},
				{"str", []byte("pkg\x00")},
				{"str_offsets", strOffsets},
				{"line_str", []byte("src\x00")},
				{"addr", addresses},
				{"rnglists", ranges},
			}
			f := makeMachoDwarfFile(t, sections, compressed)
			d, err := NewMachoWrapper(f).DWARF()
			require.NoError(t, err)
			r := d.Reader()
			e, err := r.Next()
			require.NoError(t, err)
			require.NotNil(t, e)
			require.Equal(t, "src", e.Val(dwarf.AttrCompDir))
			e, err = r.Next()
			require.NoError(t, err)
			require.NotNil(t, e)
			require.Equal(t, "pkg", e.Val(dwarf.AttrName))
			gotRanges, err := d.Ranges(e)
			require.NoError(t, err)
			require.Equal(t, [][2]uint64{{0x1234, 0x1244}}, gotRanges)
		})
	}
}

func TestMachoDWARFRejectsInvalidFileRanges(t *testing.T) {
	for _, size := range []uint64{32, ^uint64(0)} {
		f := makeMachoDwarfFile(t, []machoDwarfSection{{"abbrev", []byte{0}}}, false)
		f.Sections[0].Size = size
		_, err := NewMachoWrapper(f).DWARF()
		require.ErrorContains(t, err, "DWARF section __debug_abbrev")
	}
}

type machoDwarfSection struct {
	name string
	data []byte
}

func makeMachoDwarfFile(t *testing.T, sections []machoDwarfSection, compressed bool) *macho.File {
	t.Helper()
	const headerSize = 32
	commandSize := 72 + len(sections)*80
	fileOffset := uint32(headerSize + commandSize)
	var payload bytes.Buffer
	var headers []types.Section64
	for _, section := range sections {
		name, data := "__debug_"+section.name, section.data
		if compressed {
			name = "__zdebug_" + section.name
			var buf bytes.Buffer
			buf.WriteString("ZLIB")
			require.NoError(t, binary.Write(&buf, binary.BigEndian, uint64(len(data))))
			z := zlib.NewWriter(&buf)
			_, err := z.Write(data)
			require.NoError(t, err)
			require.NoError(t, z.Close())
			data = buf.Bytes()
		}
		h := types.Section64{Addr: 0x100000000 + uint64(payload.Len()), Size: uint64(len(data)), Offset: fileOffset + uint32(payload.Len())}
		copy(h.Name[:], name)
		copy(h.Seg[:], "__DWARF")
		headers = append(headers, h)
		payload.Write(data)
	}
	var file bytes.Buffer
	for _, value := range []uint32{0xfeedfacf, uint32(types.CPUAmd64), 3, 2, 1, uint32(commandSize), 0, 0} {
		require.NoError(t, binary.Write(&file, binary.LittleEndian, value))
	}
	// External linkers can leave DWARF's segment without a VM mapping even
	// though all of its section file offsets and payloads are valid.
	segment := types.Segment64{LoadCmd: types.LC_SEGMENT_64, Len: uint32(commandSize), Offset: uint64(fileOffset), Filesz: uint64(payload.Len()), Nsect: uint32(len(sections))}
	copy(segment.Name[:], "__DWARF")
	require.NoError(t, binary.Write(&file, binary.LittleEndian, segment))
	for _, h := range headers {
		require.NoError(t, binary.Write(&file, binary.LittleEndian, h))
	}
	file.Write(payload.Bytes())
	f, err := macho.NewFile(bytes.NewReader(file.Bytes()))
	require.NoError(t, err)
	return f
}
