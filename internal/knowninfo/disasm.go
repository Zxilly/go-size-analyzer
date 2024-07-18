//go:build !wasm

package knowninfo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/samber/lo"
	"golang.org/x/sync/semaphore"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (k *KnownInfo) Disasm() error {
	startTime := time.Now()
	slog.Info("Disassemble functions...")

	fns := k.Deps.GetFunctions()

	e, err := disasm.NewExtractor(k.Wrapper, k.Size)
	if err != nil {
		if errors.Is(err, disasm.ErrArchNotSupported) {
			slog.Warn("Disassembler not supported on this architecture")
			return nil
		}
		return err
	}

	type result struct {
		addr, size uint64
		fn         *entity.Function
	}

	resultChan := make(chan result, 32)

	resultProcess, resultDone := context.WithCancel(context.Background())

	go func() {
		defer resultDone()
		for r := range resultChan {
			ok := e.CheckAddrString(r.addr, int64(r.size))
			if !ok {
				continue
			}

			k.KnownAddr.InsertDisasm(r.addr, r.size, r.fn)
		}
	}()

	var (
		maxWorkers = runtime.GOMAXPROCS(0)
		sem        = semaphore.NewWeighted(int64(maxWorkers))
	)

	lo.ForEach(fns, func(fn *entity.Function, _ int) {
		if err := sem.Acquire(resultProcess, 1); err != nil {
			slog.Error(fmt.Sprintf("Failed to acquire semaphore: %v", err))
			return
		}

		go func() {
			defer sem.Release(1)
			candidates := e.Extract(fn.Addr, fn.Addr+fn.CodeSize)

			lo.ForEach(candidates, func(p disasm.PossibleStr, _ int) {
				resultChan <- result{
					addr: p.Addr,
					size: p.Size,
					fn:   fn,
				}
			})
		}()
	})

	if err = sem.Acquire(resultProcess, int64(maxWorkers)); err != nil {
		slog.Error(fmt.Sprintf("Failed to acquire semaphore for all workers: %v", err))
	}

	close(resultChan)

	<-resultProcess.Done()

	slog.Info(fmt.Sprintf("Disassemble functions done, took %s", time.Since(startTime)))

	return nil
}
