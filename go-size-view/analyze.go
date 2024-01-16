package go_size_view

import (
	"fmt"
	"github.com/goretk/gore"
	"io"
)

type textReader struct {
	code []byte
	pc   uint64
}

func (r textReader) ReadAt(data []byte, off int64) (n int, err error) {
	if off < 0 || uint64(off) < r.pc {
		return 0, io.EOF
	}
	d := uint64(off) - r.pc
	if d >= uint64(len(r.code)) {
		return 0, io.EOF
	}
	n = copy(data, r.code[d:])
	if n < len(data) {
		err = io.ErrUnexpectedEOF
	}
	return
}

func Analyze(file *gore.GoFile) (*Bin, error) {
	secm := extractSectionsFromGoFile(file)
	size := getFileSize(file.GetFile())
	assertSectionsSize(secm, size)

	pkgs, err := configurePackages(file, secm)
	if err != nil {
		return nil, err
	}

	bin := &Bin{
		Size:       size,
		BuildInfo:  file.BuildInfo,
		SectionMap: secm,
		Packages:   pkgs,
	}

	err = increaseSectionSizeFromSymbol(secm)
	if err != nil {
		return nil, err
	}

	for _, section := range secm.Sections {
		printPercentage := func(num1, num2 uint64) string {
			var percentage float64

			// 检查除数是否为0
			if num2 == 0 {
				percentage = 100.0
			} else {
				// 计算百分比，并保留两位小数
				percentage = float64(num1) / float64(num2) * 100
			}

			return fmt.Sprintf("%.2f%%\n", percentage)
		}

		println(section.Name, section.KnownSize, section.TotalSize, printPercentage(section.KnownSize, section.TotalSize))
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

	return bin, nil
}
