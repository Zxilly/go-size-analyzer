//go:build profiler

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

	var targets []int

	_, ci := os.LookupEnv("CI")
	if ci {
		mainProfile := os.Getenv("PROFILE")
		if mainProfile == "main" {
			targets = []int{profiler.Cpu, profiler.Mem}
		} else {
			targets = []int{profiler.Mutex, profiler.Goroutine, profiler.Block, profiler.ThreadCreate}
		}
	} else {
		targets = []int{profiler.Cpu, profiler.Mem, profiler.Mutex, profiler.Goroutine, profiler.Block, profiler.ThreadCreate}
	}

	p := profiler.New(
		profiler.Conf{
			DirPath:        outputDir,
			NoShutdownHook: true,
			MemProfileType: "heap",
		},
		targets...,
	)

	p.Start()
	defer p.Stop()
	entry()
}
