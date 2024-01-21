package disasm

import (
	"debug/pe"
	"fmt"
	go_size_view "github.com/Zxilly/go-size-view/go-size-view/tool"
)

type peWrapper struct {
	file *pe.File
}

func (p *peWrapper) text() (textStart uint64, text []byte, err error) {
	imageBase := go_size_view.GetImageBase(p.file)

	sect := p.file.Section(".text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = imageBase + uint64(sect.VirtualAddress)
	text, err = sect.Data()
	return
}

func (p *peWrapper) goarch() string {
	switch p.file.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		return "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "amd64"
	case pe.IMAGE_FILE_MACHINE_ARMNT:
		return "arm"
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return "arm64"
	default:
		return ""
	}
}
