package test

import (
	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/result"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/mmap"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func GetProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func GetTestBinPath(t *testing.T) string {
	t.Helper()

	p := filepath.Join(GetProjectRoot(), "scripts", "bins", "bin-linux-1.21-amd64")
	p, err := filepath.Abs(p)
	if err != nil {
		t.Fatalf("failed to get absolute path of %s: %v", p, err)
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Fatalf("bin not exist: %s", p)
	}

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
		SkipSymbol: true,
	})
	if err != nil {
		t.Fatalf("failed to analyze %s: %v", path, err)
	}
	return r
}
