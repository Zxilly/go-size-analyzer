//go:build !js && !wasm

package diff

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/stretchr/testify/require"
)

func TestCompiledStaticStringsRevealOffsettingChanges(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	build := func(label string, alpha, beta int) string {
		code := fmt.Sprintf("package main\nvar Alpha = %q\nvar Beta = %q\nfunc main(){println(Alpha, Beta)}\n", strings.Repeat("A", alpha), strings.Repeat("B", beta))
		require.NoError(t, os.WriteFile(src, []byte(code), 0o600))
		bin := filepath.Join(dir, label)
		cmd := exec.CommandContext(t.Context(), "go", "build", "-mod=readonly", "-p=2", "-o", bin, src)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
		return bin
	}
	before := build("before", 2048, 8192)
	after := build("after", 8192, 2048)
	a, err := autoLoadFile(before, internal.Options{})
	require.NoError(t, err)
	b, err := autoLoadFile(after, internal.Options{})
	require.NoError(t, err)
	d := newDiffResult(b, a)
	require.Equal(t, a.Size, b.Size)
	require.Empty(t, d.Packages)
	byName := map[string]diffSymbol{}
	for _, symbol := range d.Symbols {
		byName[symbol.Name] = symbol
	}
	require.Contains(t, byName, "main.Alpha.string")
	require.Contains(t, byName, "main.Beta.string")
	require.Equal(t, int64(6144), byName["main.Alpha.string"].Size.Delta)
	require.Equal(t, int64(-6144), byName["main.Beta.string"].Size.Delta)
}
