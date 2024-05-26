//go:build wasm

package internal

import "log/slog"

func (k *KnownInfo) Disasm() error {
	slog.Info("disassembler disabled for wasm")
	return nil
}
