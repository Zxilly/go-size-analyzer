//go:build !js && !wasm

package knowninfo_test

import (
	"debug/dwarf"
	"debug/elf"
	"debug/gosym"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal"
	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestCompiledFunctionPreservesPclnDwarfAndSymbolBoundaries(t *testing.T) {
	const source = `package main
//go:noinline
func target(x int) int { return x + 17 }
func main() { println(target(3)) }
`
	dir := t.TempDir()
	src, bin := filepath.Join(dir, "main.go"), filepath.Join(dir, "binary")
	require.NoError(t, os.WriteFile(src, []byte(source), 0o600))
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", bin, src)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)

	object, err := elf.Open(bin)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, object.Close()) })
	symbols, err := object.Symbols()
	require.NoError(t, err)
	var symbol elf.Symbol
	for _, candidate := range symbols {
		if candidate.Name == "main.target" {
			symbol = candidate
			break
		}
	}
	require.Positive(t, symbol.Size)

	pclnData, err := object.Section(".gopclntab").Data()
	require.NoError(t, err)
	table, err := gosym.NewTable(nil, gosym.NewLineTable(pclnData, object.Section(".text").Addr))
	require.NoError(t, err)
	physical := table.LookupFunc("main.target")
	require.NotNil(t, physical)
	require.Equal(t, symbol.Value, physical.Entry)

	debugData, err := object.DWARF()
	require.NoError(t, err)
	var declared [][2]uint64
	reader := debugData.Reader()
	for {
		entry, err := reader.Next()
		require.NoError(t, err)
		if entry == nil {
			break
		}
		if entry.Tag == dwarf.TagSubprogram && entry.Val(dwarf.AttrName) == "main.target" {
			declared, err = debugData.Ranges(entry)
			require.NoError(t, err)
			break
		}
	}
	require.Len(t, declared, 1)
	require.Equal(t, symbol.Value, declared[0][0])

	f, err := utils.OpenBinary(bin)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })
	report, err := internal.Analyze(bin, f, uint64(f.Len()), internal.Options{SkipDisasm: true, CoverageDetails: true})
	require.NoError(t, err)
	pkg := report.Packages["main"]
	require.NotNil(t, pkg)
	got := map[string][]uint64{}
	for addr := range pkg.OwnAddresses {
		if addr.Type == entity.AddrTypeText && addr.Addr == symbol.Value {
			require.Equal(t, entity.AddrRoleDefinition, addr.Role)
			got[addr.SourceType] = append(got[addr.SourceType], addr.Size)
		}
	}
	require.Equal(t, map[string][]uint64{
		entity.AddrSourceGoPclntab: {physical.End - physical.Entry},
		entity.AddrSourceSymbol:    {symbol.Size},
		entity.AddrSourceDwarf:     {declared[0][1] - declared[0][0]},
	}, got)
	for fn := range pkg.Functions {
		if fn.Addr == symbol.Value {
			require.Equal(t, "target", fn.Name)
			require.Equal(t, physical.End-physical.Entry, fn.CodeSize, "new evidence must not replace the original function boundary")
		}
	}
}
