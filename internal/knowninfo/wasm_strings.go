package knowninfo

import (
	"log/slog"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
)

func (k *KnownInfo) AnalyzeWasmStrings() {
	w, ok := k.Wrapper.(*wrapper.WasmWrapper)
	if !ok {
		return
	}
	e := disasm.NewDataExtractor(w, k.Size, w.FileDataContains, nil)
	k.KnownAddr.BuildSymbolCoverage()
	count := 0
	for fn := range k.Deps.Functions {
		body, ok := w.FunctionInstructions(fn.Addr, k.VersionFlag.Meq125)
		if !ok {
			continue
		}
		for _, p := range e.Resolve(disasm.ExtractWasm(body)) {
			if e.ValidateReference(p) {
				source := entity.AddrSourceDisasm
				if p.Copy {
					source = entity.AddrSourceStaticCopy
				}
				k.KnownAddr.InsertDisasm(p.Addr, p.Size, fn, source)
				count++
			}
		}
	}
	slog.Info("Wasm static data references analyzed", "candidates", count)
}
