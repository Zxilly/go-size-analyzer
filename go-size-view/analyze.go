package go_size_view

import (
	"fmt"
	"github.com/Zxilly/go-size-view/go-size-view/objfile"
	"github.com/goretk/gore"
	"io"
	"strings"
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
	sections := extractSectionsFromGoFile(file)
	size := getFileSize(file.GetFile())
	assertSectionsSize(sections, size)

	pkgs, err := loadPackages(file, sections)
	if err != nil {
		return nil, err
	}

	bin := &Bin{
		Size:      size,
		BuildInfo: file.BuildInfo,
		Sections:  sections,
		Packages:  pkgs,
	}

	objf, err := objfile.Create(file.GetFile(), file.GetParsedFile())
	if err != nil {
		return nil, err
	}

	disasm, err := objf.Disasm()
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs.Self {
		for _, fi := range pkg.Files {
			for _, fn := range fi.Functions {
				if !strings.Contains(fn.Name, "ConstString") {
					continue
				}
				fmt.Println("")
				fmt.Printf("name: %s following:\n", fn.Name)
				disasm.Filter(fn.Offset, fn.End)
			}
		}
	}

	return bin, nil
}
