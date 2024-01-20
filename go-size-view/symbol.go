package go_size_view

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"github.com/goretk/gore"
)

func collectSizeFromSymbol(file *gore.GoFile, b *KnownInfo) {
	switch f := file.GetParsedFile().(type) {
	case *pe.File:
		collectSizeFromPeSymbol(f, b)
	case *elf.File:
		collectSizeFromElfSymbol(f, b)
	case *macho.File:
		collectSizeFromMachoSymbol(f, b)
	default:
		panic("This should not happened :(")
	}
}

func collectSizeFromMachoSymbol(f *macho.File, b *KnownInfo) {

}

func collectSizeFromElfSymbol(f *elf.File, b *KnownInfo) {

}

func collectSizeFromPeSymbol(f *pe.File, b *KnownInfo) {

}
