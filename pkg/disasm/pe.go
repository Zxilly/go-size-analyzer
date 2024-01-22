package disasm

import (
	"debug/pe"
	"fmt"
	"github.com/Zxilly/go-size-analyzer/pkg/tool"
)

type peWrapper struct {
	file *pe.File
}

func (p *peWrapper) readAddr(addr, size uint64) ([]byte, error) {
	pf := p.file
	for _, sect := range pf.Sections {
		if uint64(sect.VirtualAddress) <= addr && addr+size <= uint64(sect.VirtualAddress+sect.VirtualSize) {
			data := make([]byte, size)
			if _, err := sect.ReadAt(data, int64(addr-uint64(sect.VirtualAddress))); err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("address not found")
}

func (p *peWrapper) text() (textStart uint64, text []byte, err error) {
	imageBase := tool.GetImageBase(p.file)

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
