//go:build !wasm

package knowninfo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (k *KnownInfo) Disasm() error {
	k.KnownAddr.BuildSymbolCoverage()

	startTime := time.Now()
	slog.Info("Disassemble functions...")

	e, err := disasm.NewExtractor(k.Wrapper, k.Size, k.Sects.IsData, k.GoStringSymbol)
	if err != nil {
		if errors.Is(err, disasm.ErrArchNotSupported) {
			slog.Warn("Disassembler not supported on this architecture")
			return err
		}
		return err
	}
	if runtimePkg, ok := k.Deps.GetPackage("runtime"); ok {
		var calls []uint64
		for fn := range runtimePkg.Functions {
			suffix := strings.TrimPrefix(fn.Name, "gcWriteBarrier")
			if len(suffix) == 1 && suffix[0] >= '1' && suffix[0] <= '8' {
				calls = append(calls, fn.Addr)
			}
		}
		e.SetWriteBarrierCalls(calls)
	}

	type result struct {
		addr, size uint64
		fn         *entity.Function
	}

	resultChan := make(chan result, 32)

	resultProcess, resultDone := context.WithCancel(context.Background())

	added := 0
	throw := 0

	go func() {
		defer resultDone()
		for r := range resultChan {
			if !e.Validate(r.addr, r.size) {
				throw++
				continue
			}
			added++

			k.KnownAddr.InsertDisasm(r.addr, r.size, r.fn)
		}
	}()

	var (
		maxWorkers = runtime.NumCPU()
		eg         = errgroup.Group{}
	)
	eg.SetLimit(maxWorkers)

	for fn := range k.Deps.Functions {
		eg.Go(func() error {
			for region := range fn.CodeRegions {
				candidates := e.Extract(region.Addr, region.Addr+region.Size)
				lo.ForEach(candidates, func(p disasm.PossibleStr, _ int) {
					resultChan <- result{
						addr: p.Addr,
						size: p.Size,
						fn:   fn,
					}
				})
			}

			return nil
		})
	}

	if err = eg.Wait(); err != nil {
		slog.Error(fmt.Sprintf("Disassemble functions failed: %v", err))
		return err
	}

	close(resultChan)
	<-resultProcess.Done()

	slog.Info(fmt.Sprintf("Disassemble functions done, took %s, added %d, throw %d", time.Since(startTime), added, throw))

	return nil
}
