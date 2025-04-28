package test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func RewritePathOnDemand(t *testing.T, path string) string {
	t.Helper()

	first := path[0]
	// is upper?
	if first >= 'A' && first <= 'Z' {
		// we assume it's a Windows environment
		n := []byte(path)

		for i, c := range n {
			if c == '/' {
				n[i] = '\\'
			}
		}
		return string(n)
	}
	return path
}

func GetProjectRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, _ := runtime.Caller(0)
	return RewritePathOnDemand(t, filepath.Join(filepath.Dir(filename), "..", ".."))
}

func GetTestBinPath(t *testing.T) string {
	t.Helper()

	testdataPath := os.Getenv("TESTDATA_PATH")
	if testdataPath != "" {
		return testdataPath
	}

	p := filepath.Join(GetProjectRoot(t), "scripts", "bins", "bin-linux-1.21-amd64")
	p, err := filepath.Abs(p)
	require.NoError(t, err)

	_, err = os.Stat(p)
	require.NoError(t, err)

	return RewritePathOnDemand(t, p)
}

func GetTestDiffBinPath(t *testing.T) string {
	t.Helper()

	p := filepath.Join(GetProjectRoot(t), "scripts", "bins", "bin-linux-1.22-amd64")
	p, err := filepath.Abs(p)
	require.NoError(t, err)

	_, err = os.Stat(p)
	require.NoError(t, err)

	return RewritePathOnDemand(t, p)
}

func GetTestJSONPath(t *testing.T) string {
	t.Helper()

	p := filepath.Join(GetProjectRoot(t), "testdata", "result.json")
	p, err := filepath.Abs(p)
	require.NoError(t, err)

	return RewritePathOnDemand(t, p)
}

func GetTestGobPath(t *testing.T) string {
	t.Helper()

	p := filepath.Join(GetProjectRoot(t), "testdata", "result.gob.gz")
	p, err := filepath.Abs(p)
	require.NoError(t, err)

	return RewritePathOnDemand(t, p)
}

func GetTestResult(t *testing.T) *result.Result {
	t.Helper()

	path := GetTestBinPath(t)

	f, err := mmap.Open(path)
	require.NoError(t, err)
	defer func(f *mmap.ReaderAt) {
		require.NoError(t, f.Close())
	}(f)

	r, err := internal.Analyze(path, f, uint64(f.Len()), internal.Options{
		SkipDwarf:  false,
		SkipDisasm: true,
		SkipSymbol: false,
	})
	require.NoError(t, err)

	return r
}
