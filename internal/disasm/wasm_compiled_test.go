//go:build !js && !wasm

package disasm_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/ZxillyFork/gore"
	"github.com/stretchr/testify/require"
)

func TestCompilerGeneratedWasmStringRepresentations(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	path := filepath.Join(dir, "program.wasm")
	require.NoError(t, os.WriteFile(source, []byte(stringProgram), 0o600))
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", path, source)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)
	f, err := utils.OpenBinary(path)
	require.NoError(t, err)
	defer f.Close()
	parsed, err := gore.OpenReader(f)
	require.NoError(t, err)
	w, ok := wrapper.NewWrapper(parsed.GetParsedFile()).(*wrapper.WasmWrapper)
	require.True(t, ok)
	require.NoError(t, w.LoadRaw(f, uint64(f.Len())))
	e := disasm.NewDataExtractor(w, uint64(f.Len()), w.FileDataContains, nil)
	table, err := parsed.PCLNTab()
	require.NoError(t, err)
	wanted := map[string][]string{
		"main.literal": {"gsa_literal_value_12345"}, "main.global": {"gsa_global_header_56789"},
		"main.boxed": {"gsa_boxed_value_PQRST"}, "main.arguments": {"gsa_first_arg_ABCDE", "gsa_second_arg_FGHIJ"},
		"main.stack": {"gsa_stack_one_UVWXY", "gsa_stack_two_ZABCD"},
	}
	for _, fn := range table.Funcs {
		strings, ok := wanted[fn.Name]
		if !ok {
			continue
		}
		body, ok := w.FunctionInstructions(fn.Entry, true)
		require.True(t, ok)
		found := map[string]bool{}
		for _, p := range e.Resolve(disasm.ExtractWasm(body)) {
			if e.Validate(p.Addr, p.Size) {
				b, err := w.ReadAddr(p.Addr, p.Size)
				require.NoError(t, err)
				found[string(b)] = true
			}
		}
		for _, text := range strings {
			require.True(t, found[text], "%s did not recover %q: %v", fn.Name, text, found)
		}
		delete(wanted, fn.Name)
	}
	require.Empty(t, wanted)
}
