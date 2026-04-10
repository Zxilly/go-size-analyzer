//go:build !js && !wasm

package knowninfo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/mmap"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/test"
)

func TestAnalyzeTypesProducesResults(t *testing.T) {
	path := test.GetTestBinPath(t)

	f, err := mmap.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	r, err := internal.Analyze(path, f, uint64(f.Len()), internal.Options{
		SkipDwarf:  false,
		SkipDisasm: true,
		SkipSymbol: false,
	})
	require.NoError(t, err)
	require.NotNil(t, r)

	// Type analyzer should have run
	assert.Contains(t, r.Analyzers, entity.AnalyzerTyp)

	runtimeTypes := findPackageByName(r.Packages, "runtime/types")
	assert.NotNil(t, runtimeTypes)
	if runtimeTypes != nil {
		assert.NotEmpty(t, runtimeTypes.Symbols)
	}

	// Total known size across all sections should be > 0
	totalKnown := uint64(0)
	for _, s := range r.Sections {
		totalKnown += s.KnownSize
	}
	assert.Greater(t, totalKnown, uint64(0))
}

func TestAnalyzeTypesPropagatesModuledataErrors(t *testing.T) {
	k := buildKnownInfoWithVersion(t, "go1.2")
	err := k.AnalyzeTypes()
	require.Error(t, err)
	require.ErrorContains(t, err, "moduledata")
}

func findPackageByName(pkgs entity.PackageMap, want string) *entity.Package {
	for _, pkg := range pkgs {
		if pkg.Name == want {
			return pkg
		}
		if found := findPackageByName(pkg.SubPackages, want); found != nil {
			return found
		}
	}
	return nil
}
