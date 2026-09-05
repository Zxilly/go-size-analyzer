package disasm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractAmd64AdjacentStringInstructions(t *testing.T) {
	// LEA -16(RIP), AX; optional NOPs; MOV $5, BX.
	lea := []byte{0x48, 0x8d, 0x05, 0xf0, 0xff, 0xff, 0xff}
	mov := []byte{0xbb, 0x05, 0, 0, 0}
	for _, nops := range [][]byte{nil, {0x90}, {0x0f, 0x1f, 0x00, 0x90}} {
		code := append(append(append([]byte{}, lea...), nops...), mov...)
		got := extractAmd64(code, 0x1000)
		require.Equal(t, []PossibleStr{{Addr: 0xff7, Size: 5}}, got)
	}
	require.Empty(t, extractAmd64(lea, 0x1000), "incomplete instruction window")
	require.Empty(t, extractAmd64([]byte{0x48, 0x8d}, 0x1000), "truncated instruction")
}

func TestBarrierRegionRequiresResolvedCallAndOnlyBufferWrites(t *testing.T) {
	// MOV (AX), CX; CALL 0x2000; MOV CX, (R11).
	code := []byte{0x48, 0x8b, 0x08, 0xe8, 0xf8, 0x0f, 0, 0, 0x49, 0x89, 0x0b}
	require.True(t, x86BarrierBlock(code, 0x1000, barrierCalls{0x2000: true}))
	require.False(t, x86BarrierBlock(code, 0x1000, nil))
	code[8] = 0x48 // The store now targets BX, outside the GC buffer.
	require.False(t, x86BarrierBlock(code, 0x1000, barrierCalls{0x2000: true}))
}

func TestStringEvidenceDoesNotCrossIndirectCalls(t *testing.T) {
	// LEA -16(RIP), BX; CALL AX; MOV $5, CX must not reuse pre-call BX.
	code := []byte{0x48, 0x8d, 0x1d, 0xf0, 0xff, 0xff, 0xff, 0xff, 0xd0, 0xb9, 5, 0, 0, 0}
	require.Empty(t, extractAmd64(code, 0x1000))
	// A partial write changes the address and must invalidate the full register.
	code = []byte{0x48, 0x8d, 0x05, 0xf0, 0xff, 0xff, 0xff, 0xb0, 1, 0xbb, 5, 0, 0, 0}
	require.Empty(t, extractAmd64(code, 0x1000))
	// DX is not the ABI length register corresponding to an AX string pointer.
	code = []byte{0x48, 0x8d, 0x05, 0xf0, 0xff, 0xff, 0xff, 0xba, 5, 0, 0, 0}
	require.Empty(t, extractAmd64(code, 0x1000))
}

func TestStringEvidenceSurvivesIndependentRegisterWrites(t *testing.T) {
	// LEA -16(RIP), AX; XOR R8D, R8D; MOV $5, BX.
	code := []byte{0x48, 0x8d, 0x05, 0xf0, 0xff, 0xff, 0xff, 0x45, 0x31, 0xc0, 0xbb, 5, 0, 0, 0}
	require.Equal(t, []PossibleStr{{Addr: 0xff7, Size: 5}}, extractAmd64(code, 0x1000))
}
