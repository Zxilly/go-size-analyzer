//go:build js && wasm

package result_test

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/gob"

	"syscall/js"
	"testing"

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

	var r result.Result
	err = gob.NewDecoder(decompressedReader).Decode(&r)
	require.NoError(t, err)

	jsValue := r.MarshalJavaScript()

	// use JSON.stringify to compare the result
	JSON := js.Global().Get("JSON")
	jsonValue := JSON.Call("stringify", jsValue).String()

	assert.JSONEq(t, testdataJSON, jsonValue)
}
