package wrapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ZxillyFork/wazero/notinternal/wasm"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

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

func TestWasmGetSectionsMarksDebugSectionsAsKnown(t *testing.T) {
	w := &WasmWrapper{
		module: &wasm.Module{
			Sections: map[string]*wasm.GenericSection{
				"code": {
					Offset: 128,
					Size:   256,
				},
				"custom_.debug_info": {
					Offset: 512,
					Size:   64,
				},
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
		}
	}

	require.NotNil(t, codeSect)
	assert.Equal(t, uint64(128), codeSect.KnownSize)
	assert.False(t, codeSect.OnlyInMemory)
	assert.False(t, codeSect.VirtualSection)

	require.NotNil(t, debugSect)
	assert.True(t, debugSect.Debug)
	assert.Equal(t, debugSect.Size, debugSect.KnownSize)
	assert.False(t, debugSect.OnlyInMemory)
	assert.False(t, debugSect.VirtualSection)
}

func TestWasmGetSectionsDataKnownSize(t *testing.T) {
	// DataSection: one active segment at offset 0x100, size 256.
	// Symbol covers [0x120, 0x140) (32 bytes, fully inside segment).
	// Expected dataSectUsed = 32.
	offsetExpr := wasm.ConstantExpression{
		Opcode: wasm.OpcodeI32Const,
		Data:   []byte{0x80, 0x02}, // leb128 i32 = 256 = 0x100
	}
	w := &WasmWrapper{
		module: &wasm.Module{
			Sections: map[string]*wasm.GenericSection{
				"data": {Offset: 8, Size: 256},
			},
			DataSection: []wasm.DataSegment{
				{OffsetExpression: offsetExpr, Init: make([]byte, 256)},
			},
		},
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
	offsetExpr := wasm.ConstantExpression{
		Opcode: wasm.OpcodeI32Const,
		Data:   []byte{0x80, 0x02}, // 256 = 0x100
	}
	w := &WasmWrapper{
		module: &wasm.Module{
			DataSection: []wasm.DataSegment{
				{OffsetExpression: offsetExpr, Init: make([]byte, 256)},
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
	offsetExpr := wasm.ConstantExpression{
		Opcode: wasm.OpcodeI32Const,
		Data:   []byte{0x80, 0x02}, // 256 = 0x100
	}
	w := &WasmWrapper{
		module: &wasm.Module{
			DataSection: []wasm.DataSegment{
				{OffsetExpression: offsetExpr, Init: make([]byte, 256)},
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
