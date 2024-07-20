//go:build profiler && !wasm

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/knadh/profiler"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func main() {
	utils.UsePanicForExit()

	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		panic("OUTPUT_DIR environment variable is not set")
	}

	targets := []int{profiler.Cpu, profiler.Mutex, profiler.Goroutine, profiler.Block, profiler.ThreadCreate, profiler.Trace}

	p := profiler.New(
		profiler.Conf{
			DirPath:        outputDir,
			NoShutdownHook: true,
		},
		targets...,
	)

	p.Start()
	defer p.Stop()

	startWriteHeapProfile(outputDir)
	defer stopWriteHeapProfile()

	pprof.Lookup("heap")

	if err := entry(); err != nil {
		utils.FatalError(err)
	}
}

var heapProfileStop context.CancelFunc

func startWriteHeapProfile(outputDir string) {
	var ctx context.Context
	ctx, heapProfileStop = context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		id := 0

		write := func() {
			id++
			path := filepath.Join(outputDir, fmt.Sprintf("mem-%d.pprof", id))
			f, err := os.Create(path)
			defer func(f *os.File) {
				err = f.Close()
				if err != nil {
					panic(err)
				}
			}(f)
			if err != nil {
				panic(err)
			}

			err = pprof.WriteHeapProfile(f)
			if err != nil {
				panic(err)
			}
		}

		for {
			select {
			case <-ctx.Done():
				write()
				return
			case <-ticker.C:
				write()
			}
		}
	}()
}

func stopWriteHeapProfile() {
	heapProfileStop()
}
