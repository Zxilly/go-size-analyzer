package internal

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/mmap"
)

func FuzzAnalyze(f *testing.F) {
	f.Fuzz(func(t *testing.T, name string, data []byte) {
		require.NotPanics(t, func() {
			reader := bytes.NewReader(data)
			_, err := Analyze(name, reader, uint64(len(data)), Options{})
			if err != nil {
				t.Logf("Error: %v", err)
			}
		})
	})
}

func GetCurrentRunningBinary(t *testing.T) string {
	t.Helper()

	path, err := os.Executable()
	require.NoError(t, err)

	return path
}

func TestAnalyzeImports(t *testing.T) {
	bin := GetCurrentRunningBinary(t)

	f, err := mmap.Open(bin)
	require.NoError(t, err)
	defer func() {
		err := f.Close()
		require.NoError(t, err)
	}()

	result, err := Analyze(bin, f, uint64(f.Len()), Options{
		SkipDisasm: true,
		SkipDwarf:  true,
		SkipSymbol: true,
		Imports:    true,
	})
	require.NoError(t, err)

	require.NotNil(t, result)

	testingPkg := result.Packages["testing"]
	require.NotNil(t, testingPkg)

	require.Contains(t, testingPkg.ImportedBy, "github.com/Zxilly/go-size-analyzer/internal")
}
