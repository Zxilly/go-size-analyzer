package tool

import (
	"debug/pe"
	"os"
)

func GetFileSize(file *os.File) uint64 {
	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}
	return uint64(fileInfo.Size())
}

func GetImageBase(file *pe.File) uint64 {
	switch hdr := file.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		return uint64(hdr.ImageBase)
	case *pe.OptionalHeader64:
		return hdr.ImageBase
	default:
		panic("unknown optional header type")
	}
}
