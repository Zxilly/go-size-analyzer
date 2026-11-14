//go:build js && wasm

package result_test

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/gob"
	"encoding/json/v2"
	"os"
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/entity/marshaler"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func TestResultMarshalJavaScriptCross(t *testing.T) {
	testdataGob, err := os.ReadFile("../../testdata/result.gob.gz")
	require.NoError(t, err)

	decompressedReader, err := gzip.NewReader(bytes.NewBuffer(testdataGob))
	require.NoError(t, err)

	r := new(result.Result)
	err = gob.NewDecoder(decompressedReader).Decode(r)
	require.NoError(t, err)

	jsonPrinterResult, err := json.Marshal(r,
		json.DefaultOptionsV2(),
		json.Deterministic(true),
		json.WithMarshalers(marshaler.GetFileDetailMarshaler()))
	require.NoError(t, err)

	resultJSAny := r.MarshalJavaScript()
	resultJSJson, err := json.Marshal(resultJSAny)
	require.NoError(t, err)

	assert.JSONEq(t, string(jsonPrinterResult), string(resultJSJson))
}

func TestResultWebBridgeOmitsRegionsWithoutMutatingFullJSON(t *testing.T) {
	r := &result.Result{Coverage: &entity.FileCoverage{
		Version: 2, Attributed: 10, BySource: map[string]uint64{"symbol": 10}, Notes: []string{"preserved"},
		Regions: []entity.FileRegion{{FileRange: entity.FileRange{Offset: 1, Size: 10}, Class: "attributed", Owners: []string{"p"}}},
	}}
	before, err := json.Marshal(r)
	require.NoError(t, err)
	native, err := json.Marshal(r, json.WithMarshalers(marshaler.GetFileDetailMarshaler()))
	require.NoError(t, err)
	browser := js.Global().Get("JSON").Call("stringify", r.MarshalJavaScript()).String()
	require.JSONEq(t, string(native), browser)
	require.NotContains(t, browser, `"regions":`)
	require.Contains(t, browser, "preserved")
	after, err := json.Marshal(r)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Contains(t, string(after), `"regions":`)
}
