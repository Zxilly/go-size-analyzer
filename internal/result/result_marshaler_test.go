//go:build js && wasm

package result_test

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/gob"
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

//go:embed testdata/result.gob.gz
var testdataGob2 []byte

func TestResultMarshalJavaScriptCross(t *testing.T) {
	decompressedReader, err := gzip.NewReader(bytes.NewBuffer(testdataGob2))
	require.NoError(t, err)

	r := new(result.Result)
	err = gob.NewDecoder(decompressedReader).Decode(r)
	require.NoError(t, err)

	jsonPrinterResult, err := json.Marshal(r,
		json.DefaultOptionsV2(),
		json.Deterministic(true),
		json.WithMarshalers(entity.FileMarshalerCompact))
	require.NoError(t, err)

	resultJSAny := r.MarshalJavaScript()
	resultJSJson, err := json.Marshal(resultJSAny)
	require.NoError(t, err)

	assert.JSONEq(t, string(jsonPrinterResult), string(resultJSJson))
}
