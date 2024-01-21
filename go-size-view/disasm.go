package go_size_view

import (
	"github.com/Zxilly/go-size-view/go-size-view/disasm"
	"github.com/goretk/gore"
	"log"
	"unicode/utf8"
)

func TryExtractWithDisasm(f *gore.GoFile, k *KnownInfo) error {
	file := f.GetFile()
	pkgs := k.Packages.GetPackages()

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
				var d = make([]byte, p.Size)
				_, err = file.ReadAt(d, int64(offset))
				if err != nil {
					log.Printf("read file failed: %v", err)
				}
				ok := utf8.Valid(d)
				if !ok {
					continue
				}
				log.Printf("possible string: %s", string(d))
			}
		}
	}

	return nil
}
