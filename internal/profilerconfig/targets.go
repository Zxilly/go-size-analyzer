package profilerconfig

import "github.com/knadh/profiler"

// Targets returns the profiler modes that are safe to enable for goos.
func Targets(goos string) []int {
	targets := []int{profiler.Cpu}
	// Mutex profiling can deadlock Windows asynchronous preemption in Go 1.27.
	// Keep it disabled until https://go.dev/cl/818940 reaches a supported release.
	if goos != "windows" {
		targets = append(targets, profiler.Mutex)
	}
	return append(targets, profiler.Goroutine, profiler.Block, profiler.ThreadCreate, profiler.Trace)
}
