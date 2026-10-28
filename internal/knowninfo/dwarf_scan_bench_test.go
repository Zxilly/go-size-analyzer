//go:build !js && !wasm

package knowninfo

import (
	"debug/dwarf"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
	"github.com/Zxilly/go-size-analyzer/internal/wrapper"
	"github.com/blacktop/go-macho"
	"github.com/stretchr/testify/require"
)

func BenchmarkTryLoadDwarfMachO(b *testing.B) {
	path := filepath.Join("..", "..", "scripts", "bins", "bin-darwin-1.27-amd64-pie-cgo")
	file, err := macho.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		b.Skip("external-link Mach-O fixture is not installed")
	}
	require.NoError(b, err)
	b.Cleanup(func() { require.NoError(b, file.Close()) })
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.Cleanup(func() { slog.SetDefault(previousLogger) })
	w := wrapper.NewMachoWrapper(file)
	d, err := w.DWARF()
	require.NoError(b, err)
	r := d.Reader()
	var entries, targets, siblings, allSiblings int
	for {
		entry, err := r.Next()
		require.NoError(b, err)
		if entry == nil {
			break
		}
		entries++
		if entry.Children && entry.Val(dwarf.AttrSibling) != nil {
			allSiblings++
		}
		if entry.Tag == dwarf.TagCompileUnit || entry.Tag == dwarf.TagSubprogram || entry.Tag == dwarf.TagVariable {
			targets++
			if entry.Children && entry.Val(dwarf.AttrSibling) != nil {
				siblings++
			}
		}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		sections := w.LoadSections()
		sections.BuildCache()
		k := &KnownInfo{Wrapper: w, Sects: sections, KnownAddr: entity.NewKnownAddr(sections)}
		k.Deps = NewDependencies(k)
		loaded, err := k.TryLoadDwarf()
		require.NoError(b, err)
		require.True(b, loaded)
	}
	b.ReportMetric(float64(entries), "DIEs/input")
	b.ReportMetric(float64(targets), "targets/input")
	b.ReportMetric(float64(siblings), "siblings/input")
	b.ReportMetric(float64(allSiblings), "all-siblings/input")
}
