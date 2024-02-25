package disasm

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"github.com/goretk/gore"
)

func buildWrapper(file *gore.GoFile) rawFileWrapper {
	switch f := file.GetParsedFile().(type) {
	case *pe.File:
		return &peWrapper{file: f}
	case *elf.File:
		return &elfWrapper{file: f}
	case *macho.File:
		return &machoWrapper{file: f}
	default:
		panic("This should not happened :(")
	}
}
