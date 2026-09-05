//go:build js && wasm

package knowninfo

import (
	"log/slog"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
)

func (k *KnownInfo) Disasm() error {
	slog.Info("disassembler disabled for wasm")
	return disasm.ErrArchNotSupported
}
