package pkg

import (
	"github.com/Zxilly/go-size-analyzer/pkg/disasm"
	"github.com/goretk/gore"
)

func TryExtractWithDisasm(f *gore.GoFile, k *KnownInfo) error {
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
				if e.AddrIsString(p.Addr, p.Size) {
					_ = k.FoundAddr.Insert(p.Addr, p.Size, pkg)
				}
			}
		}
	}

	return nil
}
