//go:build !js && !wasm

package result_test

import (
	"compress/gzip"
	"encoding/gob"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/test"
)

var update = flag.Bool("update", false, "update testdata")

func TestResultUpdateTestData(t *testing.T) {
	t.Helper()

	if !*update {
		t.Skip("update testdata is disabled")
	}

	r := test.GetTestResult(t)

	testdataJSON, err := os.OpenFile(filepath.Join("testdata", "result.json"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, testdataJSON.Close())
	}()

	indent := 2
	err = printer.JSON(r, &printer.JSONOption{
		HideDetail: true,
		Indent:     &indent,
		Writer:     testdataJSON,
	})
	require.NoError(t, err)

	testdataGob, err := os.OpenFile(filepath.Join("testdata", "result.gob.gz"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, testdataGob.Close())
	}()

	compressedWriter, err := gzip.NewWriterLevel(testdataGob, gzip.BestCompression)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, compressedWriter.Close())
	}()

	err = gob.NewEncoder(compressedWriter).Encode(r)
	require.NoError(t, err)
}
