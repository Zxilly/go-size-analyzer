//go:build js && wasm

package result

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/gob"
	"github.com/stretchr/testify/assert"

	"syscall/js"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/result.gob.gz
var testdataGob []byte

//go:embed testdata/result.json
var testdataJSON string

func TestResultMarshalJavaScript(t *testing.T) {
	decompressedReader, err := gzip.NewReader(bytes.NewReader(testdataGob))
	require.NoError(t, err)

	var r Result
	err = gob.NewDecoder(decompressedReader).Decode(&r)
	require.NoError(t, err)

	jsValue := r.MarshalJavaScript()

	// use JSON.stringify to compare the result
	JSON := js.Global().Get("JSON")
	stringify := JSON.Get("stringify")
	result := stringify.Invoke(jsValue).String()

	assert.JSONEq(t, testdataJSON, result)
}
