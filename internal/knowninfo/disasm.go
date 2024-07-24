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
	"golang.org/x/sync/errgroup"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (k *KnownInfo) Disasm() error {
	startTime := time.Now()
	slog.Info("Disassemble functions...")

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
		eg         = errgroup.Group{}
	)
	eg.SetLimit(maxWorkers)

	for fn := range k.Deps.Functions {
		eg.Go(func() error {
			candidates := e.Extract(fn.Addr, fn.Addr+fn.CodeSize)

			lo.ForEach(candidates, func(p disasm.PossibleStr, _ int) {
				resultChan <- result{
					addr: p.Addr,
					size: p.Size,
					fn:   fn,
				}
			})

			return nil
		})
	}

	if err = eg.Wait(); err != nil {
		slog.Error(fmt.Sprintf("Disassemble functions failed: %v", err))
		return err
	}

	close(resultChan)
	<-resultProcess.Done()

	slog.Info(fmt.Sprintf("Disassemble functions done, took %s", time.Since(startTime)))

	return nil
}
