package wrapper

import (
	"debug/elf"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func TestProcessElfSymbolsSkipsNoBitsSection(t *testing.T) {
	bssSect := &elf.Section{}
	bssSect.SectionHeader = elf.SectionHeader{
		Name:  ".bss",
		Type:  elf.SHT_NOBITS,
		Flags: elf.SHF_ALLOC | elf.SHF_WRITE,
		Addr:  0x1000,
		Size:  1 << 25,
	}
	dataSect := &elf.Section{}
	dataSect.SectionHeader = elf.SectionHeader{
		Name:  ".rodata",
		Type:  elf.SHT_PROGBITS,
		Flags: elf.SHF_ALLOC,
		Addr:  0x2000,
		Size:  16,
	}

	sections := []*elf.Section{bssSect, dataSect}

	var marked []string
	marker := func(name string, _ uint64, _ uint64, _ entity.AddrType) {
		marked = append(marked, name)
	}

	syms := []elf.Symbol{
		{Name: "main.a", Value: 0x1000, Size: 1 << 25, Section: 0},
		{Name: "main.b", Value: 0x2000, Size: 16, Section: 1},
	}

	err := processElfSymbols(syms, sections, marker, func(_ uint64, _ uint64) {})
	require.NoError(t, err)
	assert.Equal(t, []string{"main.b"}, marked, "bss symbol should be skipped")
}

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
