//go:build !profiler && !pgo

package main

import "github.com/Zxilly/go-size-analyzer/internal/utils"

func main() {
	if err := entry(); err != nil {
		utils.FatalError(err)
	}
}
