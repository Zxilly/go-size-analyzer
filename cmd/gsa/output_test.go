//go:build !wasm

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutputFailuresPreserveFiles(t *testing.T) {
	for _, mode := range []string{"same-input", "hardlink-input", "diff-input", "invalid-format", "duplicate-path", "invalid-binary", "unsupported-diff", "web-output", "tui-output"} {
		t.Run(mode, func(t *testing.T) {
			saved := Options
			t.Cleanup(func() { Options = saved })
			dir := t.TempDir()
			input, output := filepath.Join(dir, "input.bin"), filepath.Join(dir, "report.json")
			const original = "recoverable input"
			const report = "previous report"
			require.NoError(t, os.WriteFile(input, []byte(original), 0o600))
			require.NoError(t, os.WriteFile(output, []byte(report), 0o600))
			Options.Binary = input
			Options.Output = []string{output}
			switch mode {
			case "same-input":
				Options.Output = []string{input}
			case "hardlink-input":
				alias := filepath.Join(dir, "alias.bin")
				if err := os.Link(input, alias); err != nil {
					t.Skipf("hard links unavailable: %v", err)
				}
				Options.Output = []string{alias}
			case "diff-input":
				Options.DiffTarget = output
			case "invalid-format":
				Options.Output = []string{"json=" + output, "invalid=" + filepath.Join(dir, "other")}
			case "duplicate-path":
				Options.Output = []string{"json=" + output, "text=" + output}
			case "unsupported-diff":
				Options.DiffTarget = input
				format := "html"
				Options.Format = &format
			case "web-output":
				Options.Web = true
			case "tui-output":
				Options.Tui = true
			default:
			}
			require.Error(t, run())
			data, err := os.ReadFile(input)
			require.NoError(t, err)
			require.Equal(t, original, string(data))
			data, err = os.ReadFile(output)
			require.NoError(t, err)
			require.Equal(t, report, string(data))
			temps, err := filepath.Glob(filepath.Join(dir, ".gsa-*"))
			require.NoError(t, err)
			require.Empty(t, temps)
		})
	}
}

func TestBareDashUsesStdout(t *testing.T) {
	saved := Options
	t.Cleanup(func() { Options = saved })
	Options.Output = []string{"-"}
	specs, err := parseOutputs()
	require.NoError(t, err)
	require.NoError(t, prepareOutputs(specs))
	defer closeAll(specs)
	require.Empty(t, specs[0].path)
	require.Empty(t, specs[0].temp)
	require.NotNil(t, specs[0].writer)
}

func TestDiffReplacesReportAfterSuccess(t *testing.T) {
	saved := Options
	t.Cleanup(func() { Options = saved })
	dir := t.TempDir()
	input, output := filepath.Join(dir, "input.json"), filepath.Join(dir, "report.json")
	const data = `{"name":"example","size":20,"packages":{},"sections":[]}`
	require.NoError(t, os.WriteFile(input, []byte(data), 0o600))
	require.NoError(t, os.WriteFile(output, []byte("old report"), 0o600))
	Options.Binary = input
	Options.DiffTarget = input
	Options.Output = []string{output}
	require.NoError(t, run())
	result, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(result), `"old_name":"example"`)
	original, err := os.ReadFile(input)
	require.NoError(t, err)
	require.Equal(t, []byte(data), original)
	temps, err := filepath.Glob(filepath.Join(dir, ".gsa-*"))
	require.NoError(t, err)
	require.Empty(t, temps)
}
