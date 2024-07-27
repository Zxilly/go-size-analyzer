package wrapper

import (
	"debug/dwarf"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"errors"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

var ErrNoSymbolTable = errors.New("no symbol table found")

type RawFileWrapper interface {
	Text() (textStart uint64, text []byte, err error)
	GoArch() string
	ReadAddr(addr, size uint64) ([]byte, error)
	LoadSymbols(marker func(name string, addr, size uint64, typ entity.AddrType)) error
	LoadSections() *entity.Store
	DWARF() (*dwarf.Data, error)
}

func NewWrapper(file any) RawFileWrapper {
	switch f := file.(type) {
	case *elf.File:
		return &ElfWrapper{f}
	case *pe.File:
		return &PeWrapper{f, utils.GetImageBase(f)}
	case *macho.File:
		return &MachoWrapper{f}
	}
	return nil
}
