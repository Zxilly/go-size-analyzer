package diff

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/printer"
	"github.com/Zxilly/go-size-analyzer/internal/test"
)

func TestDiffAPIRetainsFunctionChangesAndJSONIndent(t *testing.T) {
	dir := t.TempDir()
	oldPath, newPath := filepath.Join(dir, "old.json"), filepath.Join(dir, "new.json")
	for path, r := range map[string]*commonResult{
		oldPath: functionReport(measured("f", 12, 3, 5)),
		newPath: functionReport(measured("f", 12, 3, 25)),
	} {
		data, err := json.Marshal(r)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, data, 0o600))
	}
	var output bytes.Buffer
	indent := 2
	require.NoError(t, Diff(&output, Options{OldTarget: oldPath, NewTarget: newPath, Format: "json", Indent: &indent}))
	require.Contains(t, output.String(), "\n  ")
	var result diffResult
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	require.Empty(t, result.Packages)
	require.Len(t, result.Functions, 1)
	require.Equal(t, "p.f", result.Functions[0].FullName)
	require.Equal(t, int64(20), result.Functions[0].ReferencedBytes.Delta)
}

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
			require.NoError(t, Diff(io.Discard, Options{
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

	require.Error(t, Diff(io.Discard, Options{
		OldTarget: first,
		NewTarget: second,
	}))
}

func TestFormatAnalyzer(t *testing.T) {
	cases := []struct {
		name      string
		analyzers []string
		expected  string
	}{
		{
			name:      "Empty slice returns 'none'",
			analyzers: []string{},
			expected:  "none",
		},
		{
			name:      "Single element returns the element itself",
			analyzers: []string{"Analyzer1"},
			expected:  "Analyzer1",
		},
		{
			name:      "Multiple elements returns comma-separated string",
			analyzers: []string{"Analyzer1", "Analyzer2", "Analyzer3"},
			expected:  "Analyzer1, Analyzer2, Analyzer3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatAnalyzer(tc.analyzers)
			require.Equal(t, tc.expected, result)
		})
	}
}
