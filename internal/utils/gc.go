//go:build !wasm

package utils

import (
	"runtime/debug"

	"github.com/pbnjay/memory"
)

const oneGB = 1 << 30

func ApplyMemoryLimit() {
	// memory available
	avail := memory.FreeMemory()
	use := avail / 5 * 4

	// at least we need 1GB
	limit := max(use, oneGB)
	debug.SetMemoryLimit(int64(limit))
}
