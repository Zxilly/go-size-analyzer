//go:build !js && !wasm

package disasm_test

import (
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/ZxillyFork/gore"
)

func BenchmarkExtractBinaryFunctions(b *testing.B) {
	f, err := utils.OpenBinary("../../scripts/bins/bin-linux-1.21-amd64")
	if err != nil {
		b.Skipf("benchmark fixture unavailable: %v", err)
	}
	defer f.Close()
	gf, err := gore.OpenReader(f)
	if err != nil {
		b.Fatal(err)
	}
	table, err := gf.PCLNTab()
	if err != nil {
		b.Fatal(err)
	}
	raw := wrapper.NewWrapper(gf.GetParsedFile())
	extractor, err := disasm.NewExtractor(raw, uint64(f.Len()), nil, nil)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	candidates := 0
	for b.Loop() {
		for _, fn := range table.Funcs {
			candidates += len(extractor.Extract(fn.Entry, fn.End))
		}
	}
	b.ReportMetric(float64(candidates)/float64(b.N), "candidates/op")
}
