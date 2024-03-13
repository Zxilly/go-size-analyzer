package internal

import (
	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/tool"
	"log"
	"sync"
)

func (k *KnownInfo) Disasm() error {
	log.Println("Disassemble...")

	fns := k.Packages.GetFunctions()

	pb := tool.NewPb(int64(len(fns)), "Disassembling...")

	wg := sync.WaitGroup{}
	wg.Add(len(fns))

	e, err := disasm.NewExtractor(k.wrapper, k.Size)
	if err != nil {
		return err
	}

	type result struct {
		addr, size uint64
		fn         *Function
		meta       DisasmMeta
	}

	resultChan := make(chan result, 1000)

	processWorker := func(fn *Function) {
		possible := e.Extract(fn.Addr, fn.Addr+fn.Size)
		for i, p := range possible {
			if ok := e.AddrIsString(p.Addr, int64(p.Size)); ok {
				resultChan <- result{
					addr: p.Addr,
					size: p.Size,
					fn:   fn,
					meta: DisasmMeta{
						Source: GoPclntabMeta{
							FuncName:    Deduplicate(fn.Name),
							PackageName: Deduplicate(fn.Pkg.Name),
							Type:        fn.Type, // const string, no need to intern it
							Receiver:    Deduplicate(fn.Receiver),
							Filepath:    Deduplicate(fn.Filepath),
						},
						DisasmIndex: i,
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
			k.KnownAddr.InsertDisasm(r.addr, r.size, r.fn, r.meta)
		}
	}

	go collectResult()

	for _, fn := range fns {
		go processWorker(fn)
	}

	wg.Wait()
	close(resultChan)

	// wait for collectResult to release the lock
	collectLock.Lock()
	collectLock.Unlock()

	log.Println("Disassemble done")

	return nil
}
