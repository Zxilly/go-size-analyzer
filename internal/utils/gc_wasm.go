//go:build js && wasm

package utils

import (
	"runtime/debug"
)

// WasmMemoryLimit use 2 GB memory limit
const WasmMemoryLimit = 2 * 1024 * 1024 * 1024

func ApplyMemoryLimit() {
	debug.SetMemoryLimit(WasmMemoryLimit)
}
