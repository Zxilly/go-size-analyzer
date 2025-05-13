package testutils

import (
	"path/filepath"
	"runtime"
	"testing"
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
	return RewritePathOnDemand(t, filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}
