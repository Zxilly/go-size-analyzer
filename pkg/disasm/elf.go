package disasm

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
)

type elfWrapper struct {
	file *elf.File
}

func (e *elfWrapper) text() (textStart uint64, text []byte, err error) {
	sect := e.file.Section(".text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return
}

func (e *elfWrapper) goarch() string {
	switch e.file.Machine {
	case elf.EM_386:
		return "386"
	case elf.EM_X86_64:
		return "amd64"
	case elf.EM_ARM:
		return "arm"
	case elf.EM_AARCH64:
		return "arm64"
	case elf.EM_PPC64:
		if e.file.ByteOrder == binary.LittleEndian {
			return "ppc64le"
		}
		return "ppc64"
	case elf.EM_S390:
		return "s390x"
	}
	return ""
}
