//go:build js && wasm

package entity_test

import (
	"encoding/json/v2"
	"syscall/js"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/entity/marshaler"
	"github.com/stretchr/testify/require"
)

func TestFunctionMarshalJavaScriptPreservesReferenceAvailability(t *testing.T) {
	zero, refs := uint64(0), uint64(32)
	for _, fn := range []*entity.Function{
		{Name: "legacy"},
		{Name: "zero", FullName: "example.zero", ReferencedBytes: &zero, PclnFileSize: &zero},
		{
			Name: "evidence", ReferencedBytes: &refs, ReferenceEvidenceCount: 9,
			Ranges:            []entity.AddrPos{{Addr: 100, Size: 16, Type: entity.AddrTypeText}},
			PclnSize:          entity.PclnSymbolSize{Ftab: 8, GCMaps: 8, AuxData: map[string]uint64{"inline": 16}, PCData: map[string]int{"pc": 4}},
			ReferenceEvidence: []entity.FunctionEvidence{{FileRange: entity.FileRange{Offset: 200, Size: 32}, Section: ".rodata", Source: "static_read", Role: "reference", Boundary: "instruction"}},
		},
	} {
		t.Run(fn.Name, func(t *testing.T) {
			native, err := json.Marshal(fn, json.WithMarshalers(marshaler.GetFileDetailMarshaler()))
			require.NoError(t, err)
			browser := js.Global().Get("JSON").Call("stringify", fn.MarshalJavaScript()).String()
			require.JSONEq(t, string(native), browser)
			require.NotContains(t, browser, `"pcln_size":`)
			require.NotContains(t, browser, `"ranges":`)
			require.Contains(t, browser, `"pcln_bytes":`)
		})
	}
}
