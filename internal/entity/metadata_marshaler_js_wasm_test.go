//go:build js && wasm

package entity_test

import (
	"encoding/json/v2"
	"syscall/js"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestMetadataMarshalJavaScriptOptionalNumbers(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value interface{ MarshalJavaScript() any }
	}{
		{"zero package metadata", &entity.Package{}},
		{"nonzero package metadata", &entity.Package{MetadataSize: 64}},
		{"legacy coverage", &entity.FileCoverage{}},
		{"current coverage", &entity.FileCoverage{Version: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			native, err := json.Marshal(tc.value)
			require.NoError(t, err)
			browser := js.Global().Get("JSON").Call("stringify", tc.value.MarshalJavaScript()).String()
			require.JSONEq(t, string(native), browser)
		})
	}
}
