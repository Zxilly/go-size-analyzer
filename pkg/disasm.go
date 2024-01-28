package pkg

import (
	"github.com/Zxilly/go-size-analyzer/pkg/disasm"
	"github.com/Zxilly/go-size-analyzer/pkg/tool"
	"github.com/goretk/gore"
	"log"
	"sync"
)

func (k *KnownInfo) tryDisasm(f *gore.GoFile) error {
	log.Println("Disassemble...")

	pkgs, cnt := k.Packages.GetPackageAndCountFn()

	pb := tool.NewPb(int64(cnt), "Disassembling...")

	wg := sync.WaitGroup{}
	wg.Add(cnt)

	fillLock := sync.Mutex{}

	e, err := disasm.NewExtractor(f)
	if err != nil {
		return err
	}

	type result struct {
		addr, size uint64
		pkg        *Package
		meta       DisasmMeta
	}

	resultChan := make(chan result, 1000)

	process := func(fn *gore.Function, pkg *Package, typ, receiver string) {
		possible := e.Extract(fn.Offset, fn.End)
		for i, p := range possible {
			if s, ok := e.AddrIsString(p.Addr, int64(p.Size)); ok {
				resultChan <- result{
					addr: p.Addr,
					size: p.Size,
					pkg:  pkg,
					meta: DisasmMeta{
						GoPclntabMeta: GoPclntabMeta{
							FuncName:    fn.Name,
							PackageName: pkg.Name,
							Type:        typ,
							Receiver:    receiver,
						},
						DisasmIndex:  i,
						DisasmString: s,
					},
				}
			}
		}
		_ = pb.Add(1)
		wg.Done()
	}

	fill := func() {
		fillLock.Lock()
		defer fillLock.Unlock()
		for r := range resultChan {
			_ = k.FoundAddr.Insert(r.addr, r.size, r.pkg, AddrPassDisasm, r.meta)
		}
	}

	go fill()

	for _, pkg := range pkgs {
		funcs := pkg.GetFunctions()
		for _, fn := range funcs {
			go process(fn, pkg, "function", "")
		}

		methods := pkg.GetMethods()
		for _, m := range methods {
			go process(m.Function, pkg, "method", m.Receiver)
		}
	}

	wg.Wait()
	close(resultChan)
	fillLock.Lock()
	fillLock.Unlock()

	log.Println("Disassemble done")

	return nil
}
