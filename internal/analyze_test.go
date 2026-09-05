package internal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ZxillyFork/gore"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/test/testutils"
	"github.com/Zxilly/go-size-analyzer/internal/utils"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/knowninfo"
	analysisresult "github.com/Zxilly/go-size-analyzer/internal/result"
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

	f, err := utils.OpenBinary(bin)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()

	result, err := Analyze(bin, f, uint64(f.Len()), Options{
		SkipDisasm:      true,
		SkipDwarf:       true,
		SkipSymbol:      true,
		Imports:         true,
		CoverageDetails: true,
	})
	require.NoError(t, err)

	require.NotNil(t, result)
	assertFileCoverage(t, result)

	testingPkg := result.Packages["testing"]
	require.NotNil(t, testingPkg)

	require.Contains(t, testingPkg.ImportedBy, "github.com/Zxilly/go-size-analyzer/internal")
}

func TestAnalyzeWASM(t *testing.T) {
	loc := filepath.Join(testutils.GetProjectRoot(t), "testdata", "wasm", "test.wasm")
	data, err := os.ReadFile(loc)
	require.NoError(t, err)

	b := bytes.NewReader(data)

	result, err := Analyze("test.wasm", b, uint64(len(data)), Options{CoverageDetails: true})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Packages["main"])
	assertFileCoverage(t, result)
	require.Contains(t, result.Analyzers, entity.AnalyzerTyp)
	require.Contains(t, result.Analyzers, entity.AnalyzerPclntabMeta)
	require.Positive(t, countSymbols(result.Packages))

	for _, section := range result.Sections {
		require.Falsef(t, section.OnlyInMemory, "section %s should be file-backed in wasm output", section.Name)
	}
}

func assertFileCoverage(t *testing.T, r *analysisresult.Result) {
	t.Helper()
	require.NotNil(t, r.Coverage)
	c := r.Coverage
	require.Equal(t, r.Size, c.Attributed+c.Recognized+c.Unclassified)
	require.LessOrEqual(t, c.Shared, c.Attributed)
	var offset uint64
	for _, part := range c.Regions {
		require.Equal(t, offset, part.Offset)
		require.Positive(t, part.Size)
		offset += part.Size
		require.Contains(t, []string{"attributed", "recognized", "unclassified"}, part.Class)
		if part.Class == "attributed" {
			require.NotEmpty(t, part.Owners)
		}
	}
	require.Equal(t, r.Size, offset)
}

func TestAnalyzeWASMPclntabRangesAreBounded(t *testing.T) {
	loc := filepath.Join(testutils.GetProjectRoot(t), "testdata", "wasm", "test.wasm")
	data, err := os.ReadFile(loc)
	require.NoError(t, err)

	result, err := Analyze("test.wasm", bytes.NewReader(data), uint64(len(data)), Options{})
	require.NoError(t, err)

	gf, err := gore.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)

	table, err := gf.PCLNTab()
	require.NoError(t, err)

	require.Contains(t, result.Analyzers, entity.AnalyzerPclntabMeta)
	wasmInfo, ok := gf.GetParsedFile().(gore.WasmInfo)
	require.True(t, ok)
	base := uint64(0)
	end := uint64(len(wasmInfo.Memory))
	require.NotEmpty(t, table.Funcs)
	count := 0
	var check func(entity.PackageMap)
	check = func(packages entity.PackageMap) {
		for _, pkg := range packages {
			for fn := range pkg.Functions {
				require.NotEmpty(t, fn.PclnRanges)
				count++
				for _, r := range fn.PclnRanges {
					require.GreaterOrEqual(t, r.Addr, base)
					require.LessOrEqual(t, r.Addr+r.Size, end)
				}
			}
			check(pkg.SubPackages)
		}
	}
	check(result.Packages)
	require.Greater(t, count, 100)
}

func TestAnalyzeWasmRejectsUnexpectedWrapper(t *testing.T) {
	_, _, err := analyzeWasm(&knowninfo.KnownInfo{}, Options{})
	require.EqualError(t, err, "expected WebAssembly wrapper, got <nil>")
}

func countSymbols(pkgs entity.PackageMap) int {
	total := 0
	for _, pkg := range pkgs {
		total += len(pkg.Symbols)
		total += countSymbols(pkg.SubPackages)
	}
	return total
}
