package internal

import (
	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/tool"
	"github.com/goretk/gore"
	"log"
	"sync"
)

func (k *KnownInfo) Disasm() error {
	log.Println("Disassemble...")

	pkgs, cnt := k.Packages.GetPackageAndCountFn()

	pb := tool.NewPb(int64(cnt), "Disassembling...")

	wg := sync.WaitGroup{}
	wg.Add(cnt)

	e, err := disasm.NewExtractor(k.wrapper, k.Size)
	if err != nil {
		return err
	}

	type result struct {
		addr, size uint64
		pkg        *Package
		meta       DisasmMeta
	}

	resultChan := make(chan result, 1000)

	processWorker := func(fn *gore.Function, pkg *Package, typ FuncType, receiver string) {
		possible := e.Extract(fn.Offset, fn.End)
		for i, p := range possible {
			if s, ok := e.AddrIsString(p.Addr, int64(p.Size)); ok {
				resultChan <- result{
					addr: p.Addr,
					size: p.Size,
					pkg:  pkg,
					meta: DisasmMeta{
						Source: GoPclntabMeta{
							FuncName:    Deduplicate(fn.Name),
							PackageName: Deduplicate(pkg.Name),
							Type:        typ, // const string, no need to intern it
							Receiver:    Deduplicate(receiver),
						},
						DisasmIndex:  i,
						DisasmString: Deduplicate(s),
					},
				}
			}
		}
		_ = pb.Add(1)
		wg.Done()
	}

	collectLock := sync.Mutex{}
	collectResult := func() {
		collectLock.Lock()
		defer collectLock.Unlock()
		for r := range resultChan {
			k.KnownAddr.InsertDisasm(r.addr, r.size, r.pkg, r.meta)
		}
	}

	go collectResult()

	for _, pkg := range pkgs {
		funcs := pkg.GetFunctions()
		for _, fn := range funcs {
			go processWorker(fn, pkg, FuncTypeFunction, "")
		}

		methods := pkg.GetMethods()
		for _, m := range methods {
			go processWorker(m.Function, pkg, FuncTypeMethod, m.Receiver)
		}
	}

	wg.Wait()
	close(resultChan)

	// wait for collectResult to release the lock
	collectLock.Lock()
	collectLock.Unlock()

	log.Println("Disassemble done")

	return nil
}
