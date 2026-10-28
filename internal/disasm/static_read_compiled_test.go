//go:build !js && !wasm

package disasm_test

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/disasm"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/stretchr/testify/require"
)

func TestCompilerFixedWidthStaticReads(t *testing.T) {
	const source = `package main
var Table = [4]uint32{0xdeadbeef,0xff00ff00,0xaabbccdd,0xffeeddcc}
//go:noinline
func floating() float64 { return 3.141592653589793 }
//go:noinline
func constantIndex() uint32 { return Table[2] }
//go:noinline
func dynamicIndex(i int) uint32 { return Table[i] }
func main(){println(floating(),constantIndex(),dynamicIndex(1))}
`
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			dir := t.TempDir()
			src, bin := filepath.Join(dir, "main.go"), filepath.Join(dir, "binary")
			require.NoError(t, os.WriteFile(src, []byte(source), 0o600))
			cmd := exec.CommandContext(t.Context(), "go", "build", "-o", bin, src)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", out)
			f, err := elf.Open(bin)
			require.NoError(t, err)
			defer f.Close()
			stat, err := os.Stat(bin)
			require.NoError(t, err)
			raw := wrapper.NewWrapper(f)
			sections := raw.LoadSections()
			sections.BuildCache()
			e, err := disasm.NewExtractor(raw, uint64(stat.Size()), sections.IsData, nil)
			require.NoError(t, err)
			syms, err := f.Symbols()
			require.NoError(t, err)
			byName := map[string]elf.Symbol{}
			for _, sym := range syms {
				byName[sym.Name] = sym
			}
			table := byName["main.Table"]
			require.Equal(t, uint64(16), table.Size)
			pi := make([]byte, 8)
			binary.LittleEndian.PutUint64(pi, math.Float64bits(math.Pi))
			for _, name := range []string{"floating", "constantIndex", "dynamicIndex"} {
				fn, ok := byName["main."+name]
				require.True(t, ok)
				found := false
				for _, ref := range e.Extract(fn.Value, fn.Value+fn.Size) {
					if !ref.Read || !e.ValidateReference(ref) {
						continue
					}
					data, err := raw.ReadAddr(ref.Addr, ref.Size)
					require.NoError(t, err)
					switch name {
					case "floating":
						found = found || bytes.Equal(data, pi)
					case "constantIndex":
						found = found || (ref.Addr == table.Value+8 && ref.Size == 4)
					case "dynamicIndex":
						require.False(t, ref.Addr >= table.Value && ref.Addr < table.Value+table.Size, "unknown index must not invent a fixed address")
					default:
					}
				}
				if name != "dynamicIndex" {
					require.True(t, found, "missing exact static read in %s", name)
				}
			}
		})
	}
}
