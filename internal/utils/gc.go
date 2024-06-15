//go:build !wasm

package utils

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/dustin/go-humanize"
	"github.com/pbnjay/memory"
)

const oneGB = 1 << 30

func ApplyMemoryLimit() {
	// memory available
	avail := memory.FreeMemory()
	use := avail / 5 * 4

	// at least we need 1GB
	limit := max(use, oneGB)

	slog.Debug(fmt.Sprintf("memory limit: %s", humanize.Bytes(limit)))

	debug.SetMemoryLimit(int64(limit))
}
