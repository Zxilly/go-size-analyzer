package pkg

import (
	"github.com/Zxilly/go-size-analyzer/pkg/disasm"
	"github.com/goretk/gore"
	"os"
	"unicode/utf8"
)

type stringValidator struct {
	file *os.File
}

func (s *stringValidator) Validate(offset uint64, size uint64) bool {
	var d = make([]byte, size)
	_, err := s.file.ReadAt(d, int64(offset))
	if err != nil {
		return false
	}
	return utf8.Valid(d)
}

func TryExtractWithDisasm(f *gore.GoFile, k *KnownInfo) error {
	pkgs := k.Packages.GetPackages()
	validator := &stringValidator{file: f.GetFile()}

	e, err := disasm.NewExtractor(f)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		funcs := pkg.GetFunctions()
		for _, fn := range funcs {
			possible := e.Extract(fn.Offset, fn.End)
			for _, p := range possible {
				offset := k.SectionMap.AddrToOffset(p.Addr)
				if offset == 0 {
					continue
				}
				if !validator.Validate(offset, p.Size) {
					continue
				}
			}
		}
	}

	return nil
}
