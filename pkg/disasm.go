package pkg

import (
	"fmt"
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
			for i, p := range possible {
				offset := k.SectionMap.AddrToOffset(p.Addr)
				if offset == 0 {
					continue
				}
				if e.AddrIsString(p.Addr, int64(p.Size)) {
					_ = k.FoundAddr.Insert(p.Addr, p.Size, pkg, AddrPassDisasm, fmt.Sprint(fn.PackageName, ".", fn.Name, ":", i))
				}
			}
		}
	}

	return nil
}
