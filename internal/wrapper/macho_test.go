package wrapper

import (
	"debug/macho"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoArchReturnsCorrectArchitectureString(t *testing.T) {
	tests := []struct {
		cpu      macho.Cpu
		expected string
	}{
		{macho.Cpu386, "386"},
		{macho.CpuAmd64, "amd64"},
		{macho.CpuArm, "arm"},
		{macho.CpuArm64, "arm64"},
		{macho.CpuPpc64, "ppc64"},
		{macho.Cpu(0), ""}, // Unsupported CPU type
	}

	for _, test := range tests {
		m := MachoWrapper{file: &macho.File{FileHeader: macho.FileHeader{Cpu: test.cpu}}}
		result := m.GoArch()
		assert.Equal(t, test.expected, result)
	}
}
