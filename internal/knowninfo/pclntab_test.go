//go:build !js && !wasm

package knowninfo_test

import (
	"strings"
	"testing"

	"github.com/ZxillyFork/gore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/test"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

func TestAnalyzePclntabMetaProducesResults(t *testing.T) {
	path := test.GetTestBinPath(t)

	f, err := utils.OpenBinary(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	r, err := internal.Analyze(path, f, uint64(f.Len()), internal.Options{
		SkipDwarf:  false,
		SkipDisasm: true,
		SkipSymbol: false,
	})
	require.NoError(t, err)
	require.NotNil(t, r)

	// pclntab_meta analyzer should have run
	assert.Contains(t, r.Analyzers, entity.AnalyzerPclntabMeta)
	metadata := findPackageByName(r.Packages, "runtime/pclntab")
	require.NotNil(t, metadata)
	require.NotEmpty(t, metadata.Symbols)
	gf, err := gore.OpenReader(f)
	require.NoError(t, err)
	table, err := gf.PCLNTab()
	require.NoError(t, err)
	var ftabBytes uint64
	var sum func(entity.PackageMap)
	sum = func(packages entity.PackageMap) {
		for _, pkg := range packages {
			for _, sym := range pkg.Symbols {
				if strings.HasPrefix(sym.Name, "pclntab:ftab[") {
					ftabBytes += sym.Size
				}
			}
			sum(pkg.SubPackages)
		}
	}
	sum(r.Packages)
	// The Go 1.21 fixture stores two uint32 fields per function and a final PC.
	require.Equal(t, (uint64(len(table.Funcs))*2+1)*4, ftabBytes)
}

func TestAnalyzePclntabMetaPropagatesModuledataErrors(t *testing.T) {
	k := buildKnownInfoWithVersion(t, "go1.2")
	err := k.AnalyzePclntabMeta()
	require.Error(t, err)
	require.ErrorContains(t, err, "moduledata")
}
