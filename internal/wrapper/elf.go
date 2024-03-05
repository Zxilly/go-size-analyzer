package wrapper

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
)

type ElfWrapper struct {
	file *elf.File
}

func (e *ElfWrapper) ReadAddr(addr, size uint64) ([]byte, error) {
	ef := e.file
	for _, prog := range ef.Progs {
		if prog.Type != elf.PT_LOAD {
			continue
		}
		data := make([]byte, size)
		if prog.Vaddr <= addr && addr+size-1 <= prog.Vaddr+prog.Filesz-1 {
			if _, err := prog.ReadAt(data, int64(addr-prog.Vaddr)); err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("address not found")
}

func (e *ElfWrapper) Text() (textStart uint64, text []byte, err error) {
	sect := e.file.Section(".text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return
}

func (e *ElfWrapper) GoArch() string {
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
