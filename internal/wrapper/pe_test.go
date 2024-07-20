package wrapper

import (
	"debug/pe"
	"testing"
)

func TestGoArchReturnsExpectedArchitectureString(t *testing.T) {
	tests := []struct {
		name     string
		machine  uint16
		expected string
	}{
		{"Returns386ForI386Machine", pe.IMAGE_FILE_MACHINE_I386, "386"},
		{"ReturnsAmd64ForAmd64Machine", pe.IMAGE_FILE_MACHINE_AMD64, "amd64"},
		{"ReturnsArmForArmMachine", pe.IMAGE_FILE_MACHINE_ARMNT, "arm"},
		{"ReturnsArm64ForArm64Machine", pe.IMAGE_FILE_MACHINE_ARM64, "arm64"},
		{"ReturnsEmptyStringForUnknownMachine", 0xFFFF, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockFile := &pe.File{FileHeader: pe.FileHeader{Machine: test.machine}}
			wrapper := PeWrapper{file: mockFile}

			result := wrapper.GoArch()
			if result != test.expected {
				t.Errorf("Expected %s, got %s for machine type %v", test.expected, result, test.machine)
			}
		})
	}
}
