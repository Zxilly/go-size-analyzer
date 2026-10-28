package disasm

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscardedARMAddressDoesNotBecomeStackPointer(t *testing.T) {
	var code []byte
	// ADRP XZR, .; LDR X0, [SP]; LDR X1, [SP,#8].
	for _, op := range []uint32{0x9000001f, 0xf94003e0, 0xf94007e1} {
		code = binary.LittleEndian.AppendUint32(code, op)
	}
	require.Empty(t, extractArm64(code, 4096))
}

func TestSegmentRelativeLoadsAreNotStaticFileReferences(t *testing.T) {
	// MOV RAX, FS:[RIP]; MOV RBX, FS:[RIP+8]. The segment base is unknown.
	code := []byte{0x64, 0x48, 0x8b, 0x05, 0, 0, 0, 0, 0x64, 0x48, 0x8b, 0x1d, 0, 0, 0, 0}
	require.Empty(t, extractAmd64(code, 4096))
}
