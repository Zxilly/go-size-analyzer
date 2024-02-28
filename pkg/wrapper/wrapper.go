package wrapper

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
)

type RawFileWrapper interface {
	Text() (textStart uint64, text []byte, err error)
	GoArch() string
	ReadAddr(addr, size uint64) ([]byte, error)
}

func NewWrapper(file any) RawFileWrapper {
	switch f := file.(type) {
	case *elf.File:
		return &ElfWrapper{f}
	case *pe.File:
		return &PeWrapper{f}
	case *macho.File:
		return &MachoWrapper{f}
	}
	return nil
}
