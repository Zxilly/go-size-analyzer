package internal

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ZxillyFork/gore"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal/test/testutils"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
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
		err = f.Close()
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

func TestAnalyzeWASM(t *testing.T) {
	loc := filepath.Join(testutils.GetProjectRoot(t), "testdata", "wasm", "test.wasm")
	data, err := os.ReadFile(loc)
	require.NoError(t, err)

	b := bytes.NewReader(data)

	result, err := Analyze("test.wasm", b, uint64(len(data)), Options{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Packages["main"])
	require.Contains(t, result.Analyzers, entity.AnalyzerTyp)
	require.Contains(t, result.Analyzers, entity.AnalyzerPclntabMeta)
	require.Greater(t, countSymbols(result.Packages), 0)

	for _, section := range result.Sections {
		require.Falsef(t, section.OnlyInMemory, "section %s should be file-backed in wasm output", section.Name)
	}
}

func TestAnalyzeWASMPclntabFullyAttributed(t *testing.T) {
	loc := filepath.Join(testutils.GetProjectRoot(t), "testdata", "wasm", "test.wasm")
	data, err := os.ReadFile(loc)
	require.NoError(t, err)

	result, err := Analyze("test.wasm", bytes.NewReader(data), uint64(len(data)), Options{})
	require.NoError(t, err)

	gf, err := gore.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)

	md, err := gf.Moduledata()
	require.NoError(t, err)

	require.Contains(t, result.Analyzers, entity.AnalyzerPclntabMeta)
	require.Equal(t, md.PCLNTab().Length, sumSymbolSizesWithPrefix(result.Packages, "pclntab:"))
}

func countSymbols(pkgs entity.PackageMap) int {
	total := 0
	for _, pkg := range pkgs {
		total += len(pkg.Symbols)
		total += countSymbols(pkg.SubPackages)
	}
	return total
}

func sumSymbolSizesWithPrefix(pkgs entity.PackageMap, prefix string) uint64 {
	total := uint64(0)
	for _, pkg := range pkgs {
		for _, sym := range pkg.Symbols {
			if strings.HasPrefix(sym.Name, prefix) {
				total += sym.Size
			}
		}
		total += sumSymbolSizesWithPrefix(pkg.SubPackages, prefix)
	}
	return total
}
