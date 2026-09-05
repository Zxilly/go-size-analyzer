//go:build !js && !wasm

package disasm_test

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/stretchr/testify/require"
)

const stringProgram = `package main
var Global="gsa_global_header_56789"
//go:noinline
func literal() string { return "gsa_literal_value_12345" }
//go:noinline
func pair(a int,b,c string) { println(a,b,c) }
//go:noinline
func arguments() { pair(4,"gsa_first_arg_ABCDE","gsa_second_arg_FGHIJ") }
//go:noinline
func global() string { return Global }
//go:noinline
func boxed() any { return "gsa_boxed_value_PQRST" }
//go:noinline
func consume(p *[2]string) { println(p[0],p[1]) }
//go:noinline
func stack() { a:=[2]string{"gsa_stack_one_UVWXY","gsa_stack_two_ZABCD"};consume(&a) }
func main(){println(literal(),global(),boxed());arguments();stack()}
`

func TestCompilerGeneratedStringRepresentations(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(source, []byte(stringProgram), 0o600))
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			path := filepath.Join(dir, arch)
			cmd := exec.CommandContext(t.Context(), "go", "build", "-o", path, source)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", out)
			f, err := elf.Open(path)
			require.NoError(t, err)
			defer f.Close()
			info, err := os.Stat(path)
			require.NoError(t, err)
			raw := wrapper.NewWrapper(f)
			sections := raw.LoadSections()
			sections.BuildCache()
			extractor, err := disasm.NewExtractor(raw, uint64(info.Size()), sections.IsData, nil)
			require.NoError(t, err)
			syms, err := f.Symbols()
			require.NoError(t, err)
			wanted := map[string][]string{
				"main.literal": {"gsa_literal_value_12345"}, "main.global": {"gsa_global_header_56789"},
				"main.boxed": {"gsa_boxed_value_PQRST"}, "main.arguments": {"gsa_first_arg_ABCDE", "gsa_second_arg_FGHIJ"},
				"main.stack": {"gsa_stack_one_UVWXY", "gsa_stack_two_ZABCD"},
			}
			for _, sym := range syms {
				strings, ok := wanted[sym.Name]
				if !ok {
					continue
				}
				found := map[string]bool{}
				for _, p := range extractor.Extract(sym.Value, sym.Value+sym.Size) {
					if !extractor.Validate(p.Addr, p.Size) {
						continue
					}
					data, err := raw.ReadAddr(p.Addr, p.Size)
					require.NoError(t, err)
					found[string(data)] = true
				}
				for _, text := range strings {
					require.True(t, found[text], "%s did not recover %q: %v", sym.Name, text, found)
				}
				delete(wanted, sym.Name)
			}
			require.Empty(t, wanted, "compiler fixture functions missing")
		})
	}
}
