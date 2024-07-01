//go:build js && wasm

package result_test

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/gob"
	"os"
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		json.WithMarshalers(marshaler.GetFileCompactMarshaler()))
	require.NoError(t, err)

	resultJSAny := r.MarshalJavaScript()
	resultJSJson, err := json.Marshal(resultJSAny)
	require.NoError(t, err)

	assert.JSONEq(t, string(jsonPrinterResult), string(resultJSJson))
}
