//go:build js && wasm

package utils

import (
	"runtime/debug"
)

// WasmMemoryLimit use 3 GB memory limit
const WasmMemoryLimit = 3 * 1024 * 1024 * 1024

func ApplyMemoryLimit() {
	debug.SetMemoryLimit(WasmMemoryLimit)
}
