//go:build !js && !wasm

package knowninfo_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestCompiledTypedPointersRemainReferences(t *testing.T) {
	const payload = "pointer_reachable_string_e749d764bc304659"
	const source = `package main
type Payload struct { Text string; Next *Payload }
var Shared = Payload{Text:"pointer_reachable_shared_50ab44e28385"}
var Root = &Payload{Text:"` + payload + `",Next:&Shared}
var Alias = Root
func main(){println(Root.Text,Root.Next.Text,Alias.Text)}
`
	for _, target := range []struct {
		name, goos, goarch string
		pie                bool
	}{{"linux-amd64", "linux", "amd64", false}, {"linux-pie", "linux", "amd64", true}, {"windows-amd64", "windows", "amd64", false}, {"darwin-arm64", "darwin", "arm64", false}} {
		t.Run(target.name, func(t *testing.T) {
			dir := t.TempDir()
			src, bin := filepath.Join(dir, "main.go"), filepath.Join(dir, "binary")
			require.NoError(t, os.WriteFile(src, []byte(source), 0o600))
			args := []string{"build", "-mod=readonly", "-p=2", "-o", bin}
			if target.pie {
				args = append(args, "-buildmode=pie")
			}
			args = append(args, src)
			cmd := exec.CommandContext(t.Context(), "go", args...)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOOS="+target.goos, "GOARCH="+target.goarch, "CGO_ENABLED=0", "GOMAXPROCS=4")
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			f, err := utils.OpenBinary(bin)
			require.NoError(t, err)
			defer f.Close()
			r, err := internal.Analyze(bin, f, uint64(f.Len()), internal.Options{SkipDisasm: true, CoverageDetails: true})
			require.NoError(t, err)
			pkg := r.Packages["main"]
			require.NotNil(t, pkg)
			roots := map[string]bool{}
			for a := range pkg.OwnAddressValues {
				if a.SourceType != entity.AddrSourceDwarf || a.Symbol == nil || a.Size != uint64(len(payload)) {
					continue
				}
				name := strings.TrimPrefix(a.Symbol.Name, "main.")
				if strings.HasPrefix(name, "Root.*.") {
					roots["Root"] = true
				} else if strings.HasPrefix(name, "Alias.*.") {
					roots["Alias"] = true
				} else {
					continue
				}
				require.Equal(t, entity.AddrRoleReference, a.Role)
			}
			// Identical address evidence in one package keeps one deterministic
			// name, so Alias and Root need not produce duplicate symbol records.
			require.True(t, roots["Root"] || roots["Alias"], "typed pointer must reach initialized string bytes")
			require.Equal(t, r.Size, r.Coverage.Attributed+r.Coverage.Recognized+r.Coverage.Unclassified)
		})
	}
}
