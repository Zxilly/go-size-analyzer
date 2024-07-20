package wrapper

import (
	"debug/elf"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoArchReturnsExpectedArchitectures(t *testing.T) {
	tests := []struct {
		machine      elf.Machine
		expectedArch string
		byteOrder    binary.ByteOrder
	}{
		{elf.EM_386, "386", binary.LittleEndian},
		{elf.EM_X86_64, "amd64", binary.LittleEndian},
		{elf.EM_ARM, "arm", binary.LittleEndian},
		{elf.EM_AARCH64, "arm64", binary.LittleEndian},
		{elf.EM_PPC64, "ppc64", binary.BigEndian},      // Adjusted for big endian
		{elf.EM_PPC64, "ppc64le", binary.LittleEndian}, // Explicitly little endian
		{elf.EM_S390, "s390x", binary.BigEndian},
		{0, "", binary.LittleEndian}, // Test for an unsupported machine type
	}

	for _, test := range tests {
		mockFile := new(elf.File)
		mockFile.FileHeader = elf.FileHeader{
			Machine:   test.machine,
			ByteOrder: test.byteOrder,
		}
		wrapper := ElfWrapper{file: mockFile}

		arch := wrapper.GoArch()
		assert.Equal(t, test.expectedArch, arch)
	}
}
