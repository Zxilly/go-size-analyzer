package disasm

import (
	"debug/macho"
	"fmt"
)

type machoWrapper struct {
	file *macho.File
}

func (m *machoWrapper) text() (textStart uint64, text []byte, err error) {
	sect := m.file.Section("__text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return
}

func (m *machoWrapper) goarch() string {
	switch m.file.Cpu {
	case macho.Cpu386:
		return "386"
	case macho.CpuAmd64:
		return "amd64"
	case macho.CpuArm:
		return "arm"
	case macho.CpuArm64:
		return "arm64"
	case macho.CpuPpc64:
		return "ppc64"
	}
	return ""
}
