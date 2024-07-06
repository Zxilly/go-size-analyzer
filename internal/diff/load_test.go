package diff

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/test"
)

func TestDiffJSONAndBinary(t *testing.T) {
	tests := []struct {
		name   string
		old    string
		new    string
		format string
	}{
		{
			name:   "json to binary",
			old:    test.GetTestJSONPath(t),
			new:    test.GetTestDiffBinPath(t),
			format: "json",
		},
		{
			name:   "binary to binary",
			old:    test.GetTestBinPath(t),
			new:    test.GetTestDiffBinPath(t),
			format: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			require.NoError(t, Diff(io.Discard, DOptions{
				OldTarget: tt.old,
				NewTarget: tt.new,
				Format:    tt.format,
			}))
		})
	}
}

func TestDifferentAnalyzer(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first")
	second := filepath.Join(dir, "second")

	createFile := func(name string, analyzers []entity.Analyzer) {
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, f.Close())
		}()

		r := commonResult{
			Analyzers: analyzers,
		}

		require.NoError(t, printer.JSON(r, f, &printer.JSONOption{}))
	}

	createFile(first, []entity.Analyzer{entity.AnalyzerDwarf, entity.AddrSourceSymbol})
	createFile(second, []entity.Analyzer{entity.AnalyzerDisasm})

	require.Error(t, Diff(io.Discard, DOptions{
		OldTarget: first,
		NewTarget: second,
	}))
}
