package wrapper

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

// MetadataRange describes encoded bytes in the input file, not virtual memory.
type MetadataRange struct {
	entity.FileRange
	Source string
}

// SymbolMetadata connects a native symbol's records to its defined address.
// It does not infer the size or ownership of the symbol's code or data.
type SymbolMetadata struct {
	Name   string
	Addr   uint64
	Type   entity.AddrType
	Ranges []MetadataRange
}

// FileMetadata contains recognized structures and symbol-associated records.
// Overlapping ranges are intentional; the file ledger resolves their union.
type FileMetadata struct {
	Ranges  []MetadataRange
	Symbols []SymbolMetadata
}

// LoadFileMetadata reads native metadata with physical file bounds. A malformed
// table returns no claims; callers may retain their existing section analysis.
func LoadFileMetadata(w RawFileWrapper, reader io.ReaderAt, size uint64) (FileMetadata, error) {
	r := metadataReader{reader, size}
	var result FileMetadata
	var err error
	switch f := w.(type) {
	case *ElfWrapper:
		result, err = f.loadFileMetadata(r)
	case *PeWrapper:
		result, err = f.loadFileMetadata(r)
	case *MachoWrapper:
		result, err = f.loadFileMetadata(r)
	default:
		return FileMetadata{}, nil
	}
	if err != nil {
		return FileMetadata{}, err
	}
	return result, nil
}

type metadataReader struct {
	reader io.ReaderAt
	size   uint64
}

func (r metadataReader) check(off, size uint64) error {
	if off > r.size || size > r.size-off || off > math.MaxInt64 || size > math.MaxInt64-off {
		return fmt.Errorf("metadata range [%#x, +%#x) outside file of size %#x", off, size, r.size)
	}
	return nil
}

func (r metadataReader) read(off, size uint64) ([]byte, error) {
	if err := r.check(off, size); err != nil {
		return nil, err
	}
	if size > uint64(int(^uint(0)>>1)) {
		return nil, fmt.Errorf("metadata range too large to read: %#x", size)
	}
	data := make([]byte, int(size))
	if size == 0 {
		return data, nil
	}
	if r.reader == nil {
		return nil, errors.New("missing metadata reader")
	}
	n, err := r.reader.ReadAt(data, int64(off))
	if err != nil {
		return nil, err
	}
	if n != len(data) {
		return nil, io.ErrUnexpectedEOF
	}
	return data, nil
}

func (r metadataReader) add(result *FileMetadata, off, size uint64, source string) error {
	if err := r.check(off, size); err != nil {
		return err
	}
	if size != 0 {
		result.Ranges = append(result.Ranges, metadataRange(off, size, source))
	}
	return nil
}

func metadataRange(off, size uint64, source string) MetadataRange {
	return MetadataRange{FileRange: entity.FileRange{Offset: off, Size: size}, Source: source}
}

func metadataProduct(count, stride uint64) (uint64, error) {
	if stride != 0 && count > math.MaxUint64/stride {
		return 0, fmt.Errorf("metadata table size overflows: %d records of %d bytes", count, stride)
	}
	return count * stride, nil
}

// metadataName retains the encoded terminator and any leading name decoration
// in its file range. Shared names are deliberately returned for every reference.
func metadataName(table []byte, off uint64) (string, uint64, error) {
	if off >= uint64(len(table)) {
		return "", 0, fmt.Errorf("symbol name offset %#x outside string table", off)
	}
	n := bytes.IndexByte(table[off:], 0)
	if n < 0 {
		return "", 0, fmt.Errorf("unterminated symbol name at %#x", off)
	}
	return string(table[off : off+uint64(n)]), uint64(n) + 1, nil
}

func metadataSymbolName(name string) bool {
	return name != "" && name != GoStringSymbol && !ignoreSymbols.Contains(name)
}

func metadataDebugSection(name string) bool {
	return strings.HasPrefix(name, ".debug_") || strings.HasPrefix(name, ".zdebug_") ||
		strings.HasPrefix(name, "__debug_") || strings.HasPrefix(name, "__zdebug_")
}
