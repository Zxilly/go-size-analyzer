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

func TestDwarfTypedAggregateBackingData(t *testing.T) {
	const source = `package main
type Words []uint32
var Named = Words{0x11223344,0x55667788,0x99aabbcc}
var Binary = []byte{0xff,0xfe,0xfd}
var Buffer = [8]byte{'A','B','C','D','E','F','G','H'}
var Fake = struct { array *byte; len, cap int }{&Buffer[1],3,3}
var Aggregate = struct { Nested [2][]uint16; Texts [2]string }{
 [2][]uint16{{0xff80,0xff81},{0xff82,0xff83}},
 [2]string{"first aggregate data","second aggregate data"},
}
func main(){println(Named[0],Binary[0],Aggregate.Nested[0][0],Aggregate.Texts[1],Fake.len)}
`
	for _, arch := range []string{"amd64", "s390x"} {
		t.Run(arch, func(t *testing.T) {
			dir := t.TempDir()
			src, bin := filepath.Join(dir, "main.go"), filepath.Join(dir, "binary")
			require.NoError(t, os.WriteFile(src, []byte(source), 0o600))
			cmd := exec.CommandContext(t.Context(), "go", "build", "-o", bin, src)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", out)
			f, err := utils.OpenBinary(bin)
			require.NoError(t, err)
			defer f.Close()
			gf, err := gore.OpenReader(f)
			require.NoError(t, err)
			k := &knowninfo.KnownInfo{Size: uint64(f.Len()), BuildInfo: gf.BuildInfo, Gore: gf, Wrapper: wrapper.NewWrapper(gf.GetParsedFile())}
			require.NoError(t, k.LoadSectionMap())
			k.KnownAddr = entity.NewKnownAddr(k.Sects)
			require.NoError(t, k.LoadGoreInfo(gf, false))
			loaded, err := k.TryLoadDwarf()
			require.NoError(t, err)
			require.True(t, loaded)
			k.AnalyzeStaticHeaders()
			pkg, ok := k.Deps.GetPackage("main")
			require.True(t, ok)
			wanted := map[string]uint64{
				"main.Named.[]uint32":               12,
				"main.Binary.[]uint8":               3,
				"main.Aggregate.Nested[0].[]uint16": 4,
				"main.Aggregate.Nested[1].[]uint16": 4,
				"main.Aggregate.Texts[0].string":    20,
				"main.Aggregate.Texts[1].string":    21,
			}
			for _, s := range pkg.Symbols {
				require.NotEqual(t, "main.Fake.data", s.Name, "DWARF says this is an ordinary struct, not a slice")
				if size, ok := wanted[s.Name]; ok {
					require.Equal(t, size, s.Size, s.Name)
					data, err := k.Wrapper.ReadAddr(s.Addr, s.Size)
					require.NoError(t, err)
					if s.Name == "main.Binary.[]uint8" {
						require.Equal(t, []byte{0xff, 0xfe, 0xfd}, data)
					}
					delete(wanted, s.Name)
				}
			}
			require.Empty(t, wanted)
		})
	}
}
