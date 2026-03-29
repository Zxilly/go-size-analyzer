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

func TestAnalyzePclntabMetaProducesResults(t *testing.T) {
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

	// pclntab_meta analyzer should have run
	assert.Contains(t, r.Analyzers, entity.AnalyzerPclntabMeta)
}
