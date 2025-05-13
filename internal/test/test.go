package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/Zxilly/go-size-analyzer/internal/test/testutils"
)

func getTestBinBasePath(t *testing.T) string {
	t.Helper()

	testdataPath := os.Getenv("TESTDATA_PATH")
	if testdataPath != "" {
		return testdataPath
	}

	return filepath.Join(testutils.GetProjectRoot(t), "scripts", "bins")
}

func GetTestBinPath(t *testing.T) string {
	t.Helper()

	p := filepath.Join(getTestBinBasePath(t), "bin-linux-1.21-amd64")
	p, err := filepath.Abs(p)
	require.NoError(t, err)

	_, err = os.Stat(p)
	require.NoError(t, err)

	return testutils.RewritePathOnDemand(t, p)
}

func GetTestDiffBinPath(t *testing.T) string {
	t.Helper()

	p := filepath.Join(getTestBinBasePath(t), "bin-linux-1.22-amd64")
	p, err := filepath.Abs(p)
	require.NoError(t, err)

	_, err = os.Stat(p)
	require.NoError(t, err)

	return testutils.RewritePathOnDemand(t, p)
}

func GetTestJSONPath(t *testing.T) string {
	t.Helper()

	p := filepath.Join(testutils.GetProjectRoot(t), "testdata", "result.json")
	p, err := filepath.Abs(p)
	require.NoError(t, err)

	return testutils.RewritePathOnDemand(t, p)
}

func GetTestGobPath(t *testing.T) string {
	t.Helper()

	p := filepath.Join(testutils.GetProjectRoot(t), "testdata", "result.gob.gz")
	p, err := filepath.Abs(p)
	require.NoError(t, err)

	return testutils.RewritePathOnDemand(t, p)
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
