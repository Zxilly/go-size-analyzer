//go:build pgo && !wasm

package main

import (
	"os"

	"github.com/knadh/profiler"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func main() {
	utils.UsePanicForExit()

	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		panic("OUTPUT_DIR environment variable is not set")
	}

	p := profiler.New(
		profiler.Conf{
			DirPath:        outputDir,
			NoShutdownHook: true,
		},
		profiler.Cpu,
	)

	p.Start()
	defer p.Stop()

	if err := entry(); err != nil {
		utils.FatalError(err)
	}
}
