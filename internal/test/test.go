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

func GetProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func GetTestBinPath(t *testing.T) string {
	t.Helper()

	p := filepath.Join(GetProjectRoot(), "scripts", "bins", "bin-linux-1.21-amd64")
	p, err := filepath.Abs(p)
	require.NoError(t, err)

	_, err = os.Stat(p)
	require.NoError(t, err)

	return p
}

func GetTestResult(t *testing.T) *result.Result {
	t.Helper()

	path := GetTestBinPath(t)

	f, err := mmap.Open(path)
	require.NoError(t, err)

	r, err := internal.Analyze(path, f, uint64(f.Len()), internal.Options{
		SkipDwarf:  true,
		SkipDisasm: true,
		SkipSymbol: false,
	})
	require.NoError(t, err)

	return r
}
