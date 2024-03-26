package wrapper

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"errors"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

var ErrNoSymbolTable = errors.New("no symbol table found")

type RawFileWrapper interface {
	Text() (textStart uint64, text []byte, err error)
	GoArch() string
	ReadAddr(addr, size uint64) ([]byte, error)
	LoadSymbols(marker func(name string, addr, size uint64, typ entity.AddrType) error) error
	LoadSections() map[string]*entity.Section
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
