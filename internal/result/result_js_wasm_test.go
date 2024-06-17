//go:build js && wasm

package result_test

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/gob"
	"syscall/js"
	"testing"

	"github.com/go-json-experiment/json"

	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/result.gob.gz
var testdataGob []byte

//go:embed testdata/result.json
var testdataJSON string

func TestResultMarshalJavaScript(t *testing.T) {
	decompressedReader, err := gzip.NewReader(bytes.NewReader(testdataGob))
	require.NoError(t, err)

	// use JSON.stringify to compare the result
	JSON := js.Global().Get("JSON")
	stringify := JSON.Get("stringify")

	var r result.Result
	err = gob.NewDecoder(decompressedReader).Decode(&r)
	require.NoError(t, err)

	t.Run("Result", func(t *testing.T) {
		jsVal := r.MarshalJavaScript()
		jsStr := stringify.Invoke(jsVal).String()
		assert.JSONEq(t, testdataJSON, jsStr)
	})

	var testdataJSONVal map[string]any
	err = json.Unmarshal([]byte(testdataJSON), &testdataJSONVal)
	require.NoError(t, err)

	t.Run("Section", func(t *testing.T) {
		sectionsAny := testdataJSONVal["sections"].([]any)

		for i, sect := range r.Sections {
			jsVal := sect.MarshalJavaScript()
			jsStr := stringify.Invoke(jsVal).String()

			sectAny := sectionsAny[i]
			sectStr, err := json.Marshal(sectAny)
			require.NoError(t, err)

			assert.JSONEq(t, string(sectStr), jsStr)
		}
	})

}
