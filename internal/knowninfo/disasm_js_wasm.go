//go:build js && wasm

package knowninfo

import (
	"log/slog"
)

func (k *KnownInfo) Disasm() error {
	slog.Info("disassembler disabled for wasm")
	return nil
}
