//go:build !js && !wasm

package knowninfo_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/knowninfo"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/ZxillyFork/gore"
	"github.com/stretchr/testify/require"
)

func TestDwarfEmbeddedDataByteOrder(t *testing.T) {
	const source = `package main
import "embed"
//go:embed payload.txt
var Message string
//go:embed payload.txt
var Blob []byte
//go:embed payload.txt
var Files embed.FS
func main() { println(Message, len(Blob)); data, _ := Files.ReadFile("payload.txt"); println(len(data)) }
`
	const payload = "embedded data byte-order regression"
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "payload.txt"), []byte(payload), 0600))
	for _, arch := range []string{"amd64", "s390x"} {
		t.Run(arch, func(t *testing.T) {
			path := filepath.Join(dir, arch)
			cmd := exec.Command("go", "build", "-o", path, filepath.Join(dir, "main.go"))
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", output)
			f, err := utils.OpenBinary(path)
			require.NoError(t, err)
			defer f.Close()
			gf, err := gore.OpenReader(f)
			require.NoError(t, err)
			k := &knowninfo.KnownInfo{Size: uint64(f.Len()), BuildInfo: gf.BuildInfo, Gore: gf, Wrapper: wrapper.NewWrapper(gf.GetParsedFile())}
			require.NoError(t, k.LoadSectionMap())
			k.KnownAddr = entity.NewKnownAddr(k.Sects)
			require.NoError(t, k.LoadGoreInfo(gf, false))
			require.True(t, k.TryLoadDwarf())
			pkg, ok := k.Deps.GetPackage("main")
			require.True(t, ok)
			wanted := map[string]bool{"main.Message.string": false, "main.Blob.[]uint8": false, "main.Files.embed:payload.txt.data": false}
			for _, s := range pkg.Symbols {
				if _, ok := wanted[s.Name]; ok {
					require.Equal(t, uint64(len(payload)), s.Size)
					data, err := k.Wrapper.ReadAddr(s.Addr, s.Size)
					require.NoError(t, err)
					require.Equal(t, payload, string(data))
					wanted[s.Name] = true
				}
			}
			for name, found := range wanted {
				require.True(t, found, "missing %s", name)
			}
		})
	}
}
