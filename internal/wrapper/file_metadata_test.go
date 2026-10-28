package wrapper

import (
	"bytes"
	"debug/elf"
	"debug/pe"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/blacktop/go-macho"
	"github.com/stretchr/testify/require"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func TestFileMetadataBoundsAndUnsupportedFiles(t *testing.T) {
	r := metadataReader{bytes.NewReader([]byte{1}), 1}
	for _, bounds := range [][2]uint64{{2, 0}, {0, 2}, {1, math.MaxUint64}, {math.MaxUint64, 1}} {
		_, err := r.read(bounds[0], bounds[1])
		require.Error(t, err)
	}
	_, err := metadataProduct(math.MaxUint64, 18)
	require.Error(t, err)
	_, _, err = metadataName([]byte("unterminated"), 0)
	require.Error(t, err)
	_, _, err = metadataName([]byte{0}, 1)
	require.Error(t, err)
	got, err := LoadFileMetadata(&WasmWrapper{}, nil, 0)
	require.NoError(t, err)
	require.Empty(t, got)
	got, err = LoadFileMetadata(nil, nil, 0)
	require.NoError(t, err)
	require.Empty(t, got)
}

func requireMetadataRanges(t *testing.T, got FileMetadata, data []byte) {
	t.Helper()
	ledger := entity.NewFileLedger(uint64(len(data)))
	for _, r := range got.Ranges {
		require.LessOrEqual(t, r.Offset, uint64(len(data)))
		require.LessOrEqual(t, r.Size, uint64(len(data))-r.Offset)
		require.NoError(t, ledger.Add(entity.FileClaim{FileRange: r.FileRange, Class: 1, Source: r.Source}))
	}
	for _, symbol := range got.Symbols {
		for _, r := range symbol.Ranges {
			require.NoError(t, ledger.Add(entity.FileClaim{FileRange: r.FileRange, Class: 2, Owner: symbol.Name, Source: r.Source}))
		}
	}
	coverage := ledger.Finish(true)
	require.Equal(t, uint64(len(data)), coverage.Attributed+coverage.Recognized+coverage.Unclassified)
}

func TestFileMetadataRealBinaries(t *testing.T) {
	for _, name := range []string{
		"bin-linux-1.27-amd64", "bin-linux-1.27-amd64-cgo",
		"bin-windows-1.23-amd64-cgo", "bin-windows-1.23-386",
		"bin-darwin-1.27-amd64-pie-cgo", "bin-darwin-1.18-amd64",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "scripts", "bins", name)
			f, err := os.Open(path)
			if os.IsNotExist(err) {
				t.Skip("optional compiled fixture is not installed")
			}
			require.NoError(t, err)
			defer f.Close()
			info, err := f.Stat()
			require.NoError(t, err)
			var magic [4]byte
			_, err = f.ReadAt(magic[:], 0)
			require.NoError(t, err)
			var w RawFileWrapper
			switch {
			case string(magic[:]) == "\x7fELF":
				file, err := elf.NewFile(f)
				require.NoError(t, err)
				w = NewWrapper(file)
			case string(magic[:2]) == "MZ":
				file, err := pe.NewFile(f)
				require.NoError(t, err)
				w = NewWrapper(file)
			default:
				file, err := macho.NewFile(f)
				require.NoError(t, err)
				w = NewWrapper(file)
			}
			got, err := LoadFileMetadata(w, f, uint64(info.Size()))
			require.NoError(t, err)
			require.NotEmpty(t, got.Ranges)
			require.NotEmpty(t, got.Symbols)
			existing := make(map[string]uint64)
			require.NoError(t, w.LoadSymbols(func(name string, addr, _ uint64, _ entity.AddrType) { existing[name] = addr }, func(uint64, uint64) {}))
			matched := 0
			for _, s := range got.Symbols {
				if addr, ok := existing[s.Name]; ok && addr == s.Addr {
					matched++
				}
				for _, r := range s.Ranges {
					require.LessOrEqual(t, r.Offset+r.Size, uint64(info.Size()))
				}
			}
			require.Positive(t, matched, "metadata symbols must agree with existing symbol names and addresses")
			t.Logf("%d metadata records, %d match existing symbols", len(got.Symbols), matched)
		})
	}
}

func metadataTestOrders() map[string]binary.ByteOrder {
	return map[string]binary.ByteOrder{"little": binary.LittleEndian, "big": binary.BigEndian}
}
