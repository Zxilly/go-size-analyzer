package wrapper

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliben/watgo/wasmir"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

const wasmHeader = "\x00asm\x01\x00\x00\x00"

func rawWasm(parts ...[]byte) []byte {
	data := []byte(wasmHeader)
	for _, part := range parts {
		data = append(data, part...)
	}
	return data
}

func rawWasmSection(id byte, payload []byte) []byte {
	if len(payload) >= 0x80 {
		panic("test payload is too large")
	}
	return append([]byte{id, byte(len(payload))}, payload...)
}

type failingReaderAt struct {
	data []byte
	err  error
}

func (r failingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r.data)) {
		return 0, r.err
	}
	n := copy(p, r.data[off:])
	if n < len(p) {
		return n, r.err
	}
	return n, nil
}

func TestWasmLoadRawRejectsMalformedModuleEnvelope(t *testing.T) {
	forcedReadError := errors.New("forced read failure")
	truncatedSection := rawWasm([]byte{0x01, 0x02, 0x00})

	tests := []struct {
		name    string
		reader  io.ReaderAt
		size    uint64
		wantErr string
	}{
		{
			name:    "file size overflows SectionReader",
			reader:  bytes.NewReader(nil),
			size:    math.MaxInt64 + 1,
			wantErr: "WebAssembly file is too large",
		},
		{
			name:    "truncated header",
			reader:  bytes.NewReader([]byte(wasmHeader[:4])),
			size:    4,
			wantErr: "read WebAssembly header",
		},
		{
			name:    "invalid header",
			reader:  bytes.NewReader(make([]byte, len(wasmHeader))),
			size:    uint64(len(wasmHeader)),
			wantErr: "invalid WebAssembly header",
		},
		{
			name:    "missing section id",
			reader:  bytes.NewReader([]byte(wasmHeader)),
			size:    uint64(len(wasmHeader) + 1),
			wantErr: "read WebAssembly section id",
		},
		{
			name: "overflowing section size",
			reader: bytes.NewReader(rawWasm([]byte{
				0x01, 0x80, 0x80, 0x80, 0x80, 0x10,
			})),
			size:    uint64(len(wasmHeader) + 6),
			wantErr: "WebAssembly uint32 LEB128 overflow",
		},
		{
			name:    "section exceeds declared file size",
			reader:  bytes.NewReader(rawWasm([]byte{0x01, 0x02, 0x00})),
			size:    uint64(len(wasmHeader) + 3),
			wantErr: "WebAssembly section exceeds file size",
		},
		{
			name:    "section payload read fails",
			reader:  failingReaderAt{data: truncatedSection, err: forcedReadError},
			size:    uint64(len(truncatedSection) + 1),
			wantErr: forcedReadError.Error(),
		},
		{
			name:    "section payload is truncated",
			reader:  bytes.NewReader(truncatedSection),
			size:    uint64(len(truncatedSection) + 1),
			wantErr: `WebAssembly section "type" is truncated`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WasmWrapper{}
			err := w.LoadRaw(tt.reader, tt.size)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestWasmLoadRawRejectsMalformedSectionPayloads(t *testing.T) {
	tests := []struct {
		name    string
		section []byte
		wantErr string
	}{
		{
			name:    "custom section name length is truncated",
			section: rawWasmSection(0, []byte{0x80}),
			wantErr: "read WebAssembly section",
		},
		{
			name:    "custom section name is truncated",
			section: rawWasmSection(0, []byte{0x02, 'a'}),
			wantErr: "unexpected EOF",
		},
		{
			name:    "custom name allocation is bounded by its section",
			section: rawWasmSection(0, []byte{0xff, 0xff, 0xff, 0xff, 0x0f}),
			wantErr: "unexpected EOF",
		},
		{
			name:    "code function count is truncated",
			section: rawWasmSection(10, []byte{0x80}),
			wantErr: `read WebAssembly section "code"`,
		},
		{
			name:    "function body size is truncated",
			section: rawWasmSection(10, []byte{0x01, 0x80}),
			wantErr: `read WebAssembly section "code"`,
		},
		{
			name:    "local group count is truncated",
			section: rawWasmSection(10, []byte{0x01, 0x01, 0x80}),
			wantErr: `read WebAssembly section "code"`,
		},
		{
			name:    "local count is truncated",
			section: rawWasmSection(10, []byte{0x01, 0x02, 0x01, 0x80}),
			wantErr: `read WebAssembly section "code"`,
		},
		{
			name:    "local type is missing",
			section: rawWasmSection(10, []byte{0x01, 0x02, 0x01, 0x00}),
			wantErr: `read WebAssembly section "code"`,
		},
		{
			name:    "locals exceed function body",
			section: rawWasmSection(10, []byte{0x01, 0x01, 0x01, 0x00, 0x7f}),
			wantErr: "WebAssembly function locals exceed body size",
		},
		{
			name:    "function instructions are truncated",
			section: rawWasmSection(10, []byte{0x01, 0x02, 0x00}),
			wantErr: `read WebAssembly section "code"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wasmBytes := rawWasm(tt.section)
			w := &WasmWrapper{}
			err := w.LoadRaw(bytes.NewReader(wasmBytes), uint64(len(wasmBytes)))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestWasmLoadRawRecordsKnownSectionNames(t *testing.T) {
	wasmBytes := rawWasm(
		rawWasmSection(8, nil),
		rawWasmSection(12, nil),
		rawWasmSection(13, nil),
	)
	w := &WasmWrapper{}
	require.NoError(t, w.LoadRaw(bytes.NewReader(wasmBytes), uint64(len(wasmBytes))))

	sections := w.GetSections(0, 0)
	names := make([]string, 0, len(sections))
	for _, section := range sections {
		names = append(names, section.Name)
	}
	assert.ElementsMatch(t, []string{"start", "data_count", "tag"}, names)
}

func TestWasmLoadRawPreservesSectionAndFunctionSizes(t *testing.T) {
	wasmBytes := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
		// A custom .debug_info section with two payload bytes.
		0x00, 0x0e, 0x0b, '.', 'd', 'e', 'b', 'u', 'g', '_', 'i', 'n', 'f', 'o', 0xaa, 0xbb,
		// One function: one local group (two i32s), then i32.const 0; end.
		0x0a, 0x08, 0x01, 0x06, 0x01, 0x02, 0x7f, 0x41, 0x00, 0x0b,
	}

	w := &WasmWrapper{}
	require.NoError(t, w.LoadRaw(bytes.NewReader(wasmBytes), uint64(len(wasmBytes))))

	assert.Equal(t, uint64(3), w.GetFunctionSize(funcValueOffset, true))
	assert.Equal(t, wasmSection{offset: 8, size: 16, parsed: true, originalName: ".debug_info"}, w.sections[".debug_info"])
	assert.Equal(t, wasmSection{offset: 24, size: 10, kind: 10, parsed: true, originalName: "code"}, w.sections["code"])
	code, ok := w.FunctionFileRange(funcValueOffset, true)
	require.True(t, ok)
	assert.Equal(t, entity.FileRange{Offset: 31, Size: 3}, code)

	sections := w.GetSections(3, 0)
	for _, section := range sections {
		if section.Name == ".debug_info" {
			assert.True(t, section.Debug)
			assert.Zero(t, section.KnownSize)
			return
		}
	}
	t.Fatal(".debug_info section not found")
}

func TestWasmLoadSectionsMemoryDataIsVirtual(t *testing.T) {
	w := &WasmWrapper{
		memory: make([]byte, 1<<20),
	}

	store := w.LoadSections()
	require.NotNil(t, store)

	sect := store.Sections["memory.data"]
	require.NotNil(t, sect)

	// VirtualSection=true: no file backing, excluded from file-size accounting
	// and from FindSection, but included in the data address cache.
	assert.True(t, sect.VirtualSection)
	assert.False(t, sect.OnlyInMemory)
	assert.Zero(t, sect.FileSize)

	// After BuildCache, linear-memory addresses must be queryable via IsData.
	store.BuildCache()
	assert.True(t, store.IsData(0x100, 0x20))

	// Virtual sections must not appear in FindSection results (no file backing).
	assert.Nil(t, store.FindSection(0x100, 0x20))
}

func TestWasmGetSectionsKeepsDebugSectionsVisible(t *testing.T) {
	w := &WasmWrapper{
		sections: map[string]wasmSection{
			"code": {
				offset: 128,
				size:   256,
			},
			"custom_.debug_info": {
				offset: 512,
				size:   64,
			},
		},
	}

	sections := w.GetSections(128, 0)

	var codeSect, debugSect *entity.Section
	for _, section := range sections {
		switch section.Name {
		case "code":
			codeSect = section
		case "custom_.debug_info":
			debugSect = section
		default:
		}
	}

	require.NotNil(t, codeSect)
	assert.Equal(t, uint64(128), codeSect.KnownSize)
	assert.False(t, codeSect.OnlyInMemory)
	assert.False(t, codeSect.VirtualSection)

	require.NotNil(t, debugSect)
	assert.True(t, debugSect.Debug)
	assert.Zero(t, debugSect.KnownSize)
	assert.False(t, debugSect.OnlyInMemory)
	assert.False(t, debugSect.VirtualSection)
}

func TestWasmGetSectionsDataKnownSize(t *testing.T) {
	// DataSection: one active segment at offset 0x100, size 256.
	// Symbol covers [0x120, 0x140) (32 bytes, fully inside segment).
	// Expected dataSectUsed = 32.
	offsetExpr := []wasmir.Instruction{{Kind: wasmir.InstrI32Const, I32Const: 0x100}}
	w := &WasmWrapper{
		module: &wasmir.Module{
			Data: []wasmir.DataSegment{
				{Mode: wasmir.DataSegmentModeActive, OffsetExpr: offsetExpr, Init: make([]byte, 256)},
			},
		},
		sections: map[string]wasmSection{"data": {offset: 8, size: 256}},
	}

	symbols := entity.AddrSpace{}
	sym := &entity.Addr{
		AddrPos: &entity.AddrPos{Addr: 0x120, Size: 32, Type: entity.AddrTypeData},
	}
	symbols[0x120] = sym

	dataSectUsed := w.ComputeDataSectUsed(symbols)
	assert.Equal(t, uint64(32), dataSectUsed)

	sections := w.GetSections(0, dataSectUsed)
	var dataSect *entity.Section
	for _, s := range sections {
		if s.Name == "data" {
			dataSect = s
			break
		}
	}
	require.NotNil(t, dataSect)
	assert.Equal(t, uint64(32), dataSect.KnownSize)
}

func TestWasmComputeDataSectUsedExcludesZeroInit(t *testing.T) {
	// Segment covers [0x100, 0x200). Symbol at [0x50, 0x80) is outside
	// any segment (zero-initialized pages) — must not be counted.
	offsetExpr := []wasmir.Instruction{{Kind: wasmir.InstrI32Const, I32Const: 0x100}}
	w := &WasmWrapper{
		module: &wasmir.Module{
			Data: []wasmir.DataSegment{
				{Mode: wasmir.DataSegmentModeActive, OffsetExpr: offsetExpr, Init: make([]byte, 256)},
			},
		},
	}

	symbols := entity.AddrSpace{}
	// Symbol outside segment — should contribute 0.
	symbols[0x50] = &entity.Addr{
		AddrPos: &entity.AddrPos{Addr: 0x50, Size: 48, Type: entity.AddrTypeData},
	}
	// Symbol inside segment — should contribute 16.
	symbols[0x110] = &entity.Addr{
		AddrPos: &entity.AddrPos{Addr: 0x110, Size: 16, Type: entity.AddrTypeData},
	}

	got := w.ComputeDataSectUsed(symbols)
	assert.Equal(t, uint64(16), got)
}

func TestWasmComputeDataSectUsedMergesOverlappingSymbols(t *testing.T) {
	offsetExpr := []wasmir.Instruction{{Kind: wasmir.InstrI32Const, I32Const: 0x100}}
	w := &WasmWrapper{
		module: &wasmir.Module{
			Data: []wasmir.DataSegment{
				{Mode: wasmir.DataSegmentModeActive, OffsetExpr: offsetExpr, Init: make([]byte, 256)},
			},
		},
	}

	symbols := entity.AddrSpace{}
	symbols[0x110] = &entity.Addr{
		AddrPos: &entity.AddrPos{Addr: 0x110, Size: 0x40, Type: entity.AddrTypeData},
	}
	symbols[0x140] = &entity.Addr{
		AddrPos: &entity.AddrPos{Addr: 0x140, Size: 0x20, Type: entity.AddrTypeData},
	}

	// Unique coverage is [0x110, 0x160), not 0x40+0x20.
	got := w.ComputeDataSectUsed(symbols)
	assert.Equal(t, uint64(0x50), got)
}
