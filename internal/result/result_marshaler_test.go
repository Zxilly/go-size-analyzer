//go:build test_js_marshaler

package result_test

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func TestResult_MarshalJavaScript(t *testing.T) {
	testdataGob, err := os.Open(filepath.Join("testdata", "result.gob.gz"))
	require.NoError(t, err)

	decompressedReader, err := gzip.NewReader(testdataGob)
	require.NoError(t, err)

	r := new(result.Result)
	err = gob.NewDecoder(decompressedReader).Decode(r)
	require.NoError(t, err)

	jsonPrinterResult := new(bytes.Buffer)
	err = printer.JSON(r, &printer.JSONOption{
		HideDetail: true,
		Writer:     jsonPrinterResult,
	})
	require.NoError(t, err)

	resultJSAny := r.MarshalJavaScript()
	resultJSJson, err := json.Marshal(resultJSAny)
	require.NoError(t, err)

	assert.JSONEq(t, jsonPrinterResult.String(), string(resultJSJson))
}
