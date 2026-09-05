//go:build !js && !wasm

package knowninfo_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestWritableSymbolsWithoutDWARF(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	bin := filepath.Join(dir, "program")
	require.NoError(t, os.WriteFile(source, []byte("package main\nvar Global = \"writable symbol regression\"\nfunc main(){println(Global)}\n"), 0o600))
	cmd := exec.CommandContext(t.Context(), "go", "build", "-ldflags=-w", "-o", bin, source)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)
	f, err := utils.OpenBinary(bin)
	require.NoError(t, err)
	defer f.Close()
	r, err := internal.Analyze(bin, f, uint64(f.Len()), internal.Options{SkipDwarf: true, SkipDisasm: true})
	require.NoError(t, err)
	pkg := findPackageByName(r.Packages, "main")
	require.NotNil(t, pkg)
	foundHeader, foundData := false, false
	for _, sym := range pkg.Symbols {
		if sym.Name == "main.Global" {
			require.Equal(t, uint64(16), sym.Size)
			foundHeader = true
		}
		if sym.Name == "main.Global.data" {
			require.Equal(t, uint64(len("writable symbol regression")), sym.Size)
			foundData = true
		}
	}
	require.True(t, foundHeader, "writable global omitted from the symbol-only analysis")
	require.True(t, foundData, "static payload should be recovered without DWARF or disassembly")
}
