package go_size_view

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"os"
)

func getFileSize(file *os.File) uint64 {
	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}
	return uint64(fileInfo.Size())
}

func getimageBase(file *pe.File) uint64 {
	switch hdr := file.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		return uint64(hdr.ImageBase)
	case *pe.OptionalHeader64:
		return hdr.ImageBase
	default:
		panic("This should not happened :(")
	}
}

const GoArchX86 = "386"
const GoArchX64 = "amd64"

// only GoArchX86 and GoArchX64 get supported right now
func getGoArch(file any) string {
	switch f := file.(type) {
	case *pe.File:
		switch f.Machine {
		case pe.IMAGE_FILE_MACHINE_I386:
			return GoArchX86
		case pe.IMAGE_FILE_MACHINE_AMD64:
			return GoArchX64
		default:
			return ""
		}
	case *elf.File:
		switch f.Machine {
		case elf.EM_386:
			return GoArchX86
		case elf.EM_X86_64:
			return GoArchX64
		default:
			return ""
		}
	case *macho.File:
		switch f.Cpu {
		case macho.Cpu386:
			return GoArchX86
		case macho.CpuAmd64:
			return GoArchX64
		default:
			return ""
		}
	}
	return ""
}
