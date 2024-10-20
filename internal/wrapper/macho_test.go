package wrapper

import (
	"testing"

	"github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/types"

	"github.com/stretchr/testify/assert"
)

func TestGoArchReturnsCorrectArchitectureString(t *testing.T) {
	tests := []struct {
		cpu      types.CPU
		expected string
	}{
		{types.CPUI386, "386"},
		{types.CPUAmd64, "amd64"},
		{types.CPUArm, "arm"},
		{types.CPUArm64, "arm64"},
		{types.CPUPpc64, "ppc64"},
		{types.CPU(0), ""}, // Unsupported CPU type
	}

	for _, test := range tests {
		m := MachoWrapper{file: &macho.File{FileTOC: macho.FileTOC{FileHeader: types.FileHeader{CPU: test.cpu}}}}
		result := m.GoArch()
		assert.Equal(t, test.expected, result)
	}
}
