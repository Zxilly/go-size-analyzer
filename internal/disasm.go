package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func (k *KnownInfo) Disasm(gcRate int) error {
	gcRate = max(gcRate, 128)

	startTime := time.Now()
	slog.Info("Disassemble functions...")

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
			s, ok := e.LoadAddrString(r.addr, int64(r.size))
			if !ok {
				continue
			}

			k.KnownAddr.InsertDisasm(r.addr, r.size, r.fn, entity.DisasmMeta{Value: utils.Deduplicate(s)})

			processed++

			if processed%gcRate == 1 {
				// maybe too strict, but we don't have a valid memory limit
				runtime.GC()
			}
		}

		resultDone()
	}()

	numCores := runtime.NumCPU()
	disasmLimit := make(chan struct{}, numCores)
	for range numCores {
		disasmLimit <- struct{}{}
	}

	lop.ForEach(fns, func(fn *entity.Function, _ int) {
		<-disasmLimit

		candidates := e.Extract(fn.Addr, fn.Addr+fn.CodeSize)

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

	slog.Info(fmt.Sprintf("Disassemble functions done, took %s", time.Since(startTime)))

	return nil
}
