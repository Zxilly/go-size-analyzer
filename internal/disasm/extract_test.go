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
