package pkg

import (
	"github.com/Zxilly/go-size-analyzer/pkg/disasm"
	"github.com/goretk/gore"
)

func tryExtractWithDisasm(f *gore.GoFile, k *KnownInfo) error {
	pkgs := k.Packages.GetPackages()

	e, err := disasm.NewExtractor(f)
	if err != nil {
		return err
	}

	process := func(fn *gore.Function, pkg *Package, typ, receiver string) {
		possible := e.Extract(fn.Offset, fn.End)
		for i, p := range possible {
			if s, ok := e.AddrIsString(p.Addr, int64(p.Size)); ok {
				_ = k.FoundAddr.Insert(p.Addr, p.Size, pkg, AddrPassDisasm, DisasmMeta{
					GoPclntabMeta: GoPclntabMeta{
						FuncName:    fn.Name,
						PackageName: pkg.Name,
						Type:        typ,
						Receiver:    receiver,
					},
					DisasmIndex:  i,
					DisasmString: s,
				})
			}
		}
	}

	for _, pkg := range pkgs {
		funcs := pkg.GetFunctions()
		for _, fn := range funcs {
			process(fn, pkg, "function", "")
		}
		methods := pkg.GetMethods()
		for _, m := range methods {
			process(m.Function, pkg, "method", m.Receiver)
		}
	}

	return nil
}
