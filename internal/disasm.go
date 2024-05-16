package internal

import (
	"context"
	"errors"
	"log/slog"
	"runtime"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func (k *KnownInfo) Disasm() error {
	fns := k.Deps.GetFunctions()

	e, err := disasm.NewExtractor(k.wrapper, k.Size)
	if err != nil {
		if errors.Is(err, disasm.ErrArchNotSupported) {
			slog.Warn("Warning: disassembler not supported for this architecture")
			return nil
		}
		return err
	}

	type result struct {
		addr, size uint64
		fn         *entity.Function
	}

	resultChan := make(chan result, 1024)

	resultProcess, resultDone := context.WithCancel(context.Background())

	go func() {
		processed := 0

		for r := range resultChan {
			s, _ := e.LoadAddrString(r.addr, int64(r.size))
			k.KnownAddr.InsertDisasm(r.addr, r.size, r.fn, entity.DisasmMeta{Value: utils.Deduplicate(s)})

			processed++

			if processed%128 == 1 {
				// maybe too strict, but we don't have a valid memory limit
				runtime.GC()
			}
		}

		resultDone()
	}()

	slog.Info("Disassemble functions...")

	numCores := runtime.NumCPU()
	disasmLimit := make(chan struct{}, numCores)
	for range numCores {
		disasmLimit <- struct{}{}
	}

	lop.ForEach(fns, func(fn *entity.Function, _ int) {
		<-disasmLimit

		candidates := e.Extract(fn.Addr, fn.Addr+fn.CodeSize)
		candidates = lo.Filter(candidates, func(p disasm.PossibleStr, _ int) bool {
			if p.Size <= 2 {
				return false
			}

			_, ok := e.LoadAddrString(p.Addr, int64(p.Size))
			return ok
		})

		lo.ForEach(candidates, func(p disasm.PossibleStr, _ int) {
			resultChan <- result{
				addr: p.Addr,
				size: p.Size,
				fn:   fn,
			}
		})

		disasmLimit <- struct{}{}
	})

	close(resultChan)

	<-resultProcess.Done()

	slog.Info("Disassemble done")

	return nil
}
