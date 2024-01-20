package go_size_view

import (
	"github.com/goretk/gore"
	"log"
)

func Analyze(path string) error {
	file, err := gore.Open(path)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	target := &KnownInfo{}
	err = target.Collect(file)
	if err != nil {
		return err
	}

	//disasm, err := objf.Disasm()
	//if err != nil {
	//	return nil, err
	//}

	//for _, pkg := range pkgs.Self {
	//	for _, fi := range pkg.Files {
	//		for _, fn := range fi.Functions {
	//			if !strings.Contains(fn.Name, "Struct") {
	//				continue
	//			}
	//			fmt.Println("")
	//			fmt.Printf("name: %s following:\n", fn.Name)
	//			found := disasm.Filter(fn.Offset, fn.End)
	//
	//			raw := file.GetFile()
	//			elfFile := file.GetParsedFile().(*elf.File)
	//			roData := elfFile.Section(".rodata")
	//			data := elfFile.Section(".data")
	//
	//			for _, f := range found {
	//				var addr uint64
	//				var offset uint64
	//
	//				switch f.Location {
	//				case objfile.SectionData:
	//					addr = data.Addr
	//					offset = data.Offset
	//				case objfile.SectionRoData:
	//					addr = roData.Addr
	//					offset = roData.Offset
	//				default:
	//					continue
	//				}
	//
	//				fmt.Printf("start: %#x len: %#x\n", f.Start, f.Len)
	//				off := f.Start - addr + offset
	//				data := make([]byte, f.Len)
	//				_, err := raw.ReadAt(data, int64(off))
	//				if err != nil {
	//					return nil, err
	//				}
	//				println(string(data))
	//			}
	//		}
	//	}
	//}

	return nil
}
