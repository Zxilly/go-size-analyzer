package disasm

import "testing"

func TestExtractAmd64TruncatedVEX(t *testing.T) {
	// Stripped Mach-O function ranges can end with an incomplete VEX prefix.
	for _, code := range [][]byte{{0xc5, 0xed}, {0xc4, 0xe2, 0x7d}} {
		if got := extractAmd64(code, 0x1000); len(got) != 0 {
			t.Fatalf("truncated instruction produced string candidates: %v", got)
		}
	}
}
