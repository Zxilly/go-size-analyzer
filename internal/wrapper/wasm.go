package wrapper

import (
	"bufio"
	"bytes"
	"cmp"
	"debug/dwarf"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/eliben/watgo/wasmir"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type WasmWrapper struct {
	module        *wasmir.Module
	memory        []byte
	sections      map[string]wasmSection
	functionSizes []uint64
}

type wasmSection struct {
	offset uint64
	size   uint64
}

var _ RawFileWrapper = (*WasmWrapper)(nil)

const funcValueOffset = 0x1000

func (w *WasmWrapper) GetFunctionSize(idx uint64, meq125 bool) uint64 {
	// Go 1.25+ stores PC_F directly in pclntab, older versions store full PC (PC_F << 16)
	if !meq125 {
		idx = idx >> 16
	}
	// malformed pclntab entry
	if idx < funcValueOffset {
		return 0
	}
	idx -= funcValueOffset
	if idx >= uint64(len(w.functionSizes)) {
		return 0
	}

	return w.functionSizes[idx]
}

func readWasmUint32(r io.ByteReader) (uint32, uint64, error) {
	var value uint32
	for i := 0; ; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, uint64(i), err
		}
		if i == 4 && b&0xf0 != 0 {
			return 0, uint64(i + 1), errors.New("WebAssembly uint32 LEB128 overflow")
		}
		value |= uint32(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return value, uint64(i + 1), nil
		}
	}
}

func wasmSectionName(id byte) string {
	switch id {
	case 1:
		return "type"
	case 2:
		return "import"
	case 3:
		return "function"
	case 4:
		return "table"
	case 5:
		return "memory"
	case 6:
		return "global"
	case 7:
		return "export"
	case 8:
		return "start"
	case 9:
		return "element"
	case 10:
		return "code"
	case 11:
		return "data"
	case 12:
		return "data_count"
	case 13:
		return "tag"
	default:
		return fmt.Sprintf("section_%d", id)
	}
}

func readCustomSectionName(r *bufio.Reader) (string, error) {
	nameSize, _, err := readWasmUint32(r)
	if err != nil {
		return "", err
	}
	name := make([]byte, nameSize)
	if _, err = io.ReadFull(r, name); err != nil {
		return "", err
	}
	return string(name), nil
}

func readCodeFunctionSizes(r *bufio.Reader) ([]uint64, error) {
	count, _, err := readWasmUint32(r)
	if err != nil {
		return nil, err
	}

	sizes := make([]uint64, 0, count)
	for range count {
		bodySize, _, err := readWasmUint32(r)
		if err != nil {
			return nil, err
		}

		localGroupCount, localBytes, err := readWasmUint32(r)
		if err != nil {
			return nil, err
		}
		for range localGroupCount {
			_, n, err := readWasmUint32(r)
			if err != nil {
				return nil, err
			}
			localBytes += n
			if _, err = r.ReadByte(); err != nil {
				return nil, err
			}
			localBytes++
		}
		if localBytes > uint64(bodySize) {
			return nil, errors.New("WebAssembly function locals exceed body size")
		}

		instructionSize := uint64(bodySize) - localBytes
		if _, err = io.CopyN(io.Discard, r, int64(instructionSize)); err != nil {
			return nil, err
		}
		sizes = append(sizes, instructionSize)
	}
	return sizes, nil
}

// LoadRaw records file-backed section boundaries and encoded function sizes.
// watgo exposes a semantic IR, so these byte-level details must be read from
// the original module rather than inferred from decoded instructions.
func (w *WasmWrapper) LoadRaw(reader io.ReaderAt, size uint64) error {
	if size > math.MaxInt64 {
		return errors.New("WebAssembly file is too large")
	}

	r := bufio.NewReader(io.NewSectionReader(reader, 0, int64(size)))
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return fmt.Errorf("read WebAssembly header: %w", err)
	}
	if !bytes.Equal(header[:4], []byte{'\x00', 'a', 's', 'm'}) ||
		!bytes.Equal(header[4:], []byte{'\x01', '\x00', '\x00', '\x00'}) {
		return errors.New("invalid WebAssembly header")
	}

	w.sections = make(map[string]wasmSection)
	w.functionSizes = nil
	offset := uint64(len(header))
	for offset < size {
		sectionOffset := offset
		sectionID, err := r.ReadByte()
		if err != nil {
			return fmt.Errorf("read WebAssembly section id: %w", err)
		}
		offset++

		payloadSize, sizeBytes, err := readWasmUint32(r)
		if err != nil {
			return fmt.Errorf("read WebAssembly section size: %w", err)
		}
		offset += sizeBytes
		if offset > size || uint64(payloadSize) > size-offset {
			return errors.New("WebAssembly section exceeds file size")
		}

		name := wasmSectionName(sectionID)
		limited := &io.LimitedReader{R: r, N: int64(payloadSize)}
		sectionReader := bufio.NewReader(limited)
		switch sectionID {
		case 0:
			name, err = readCustomSectionName(sectionReader)
		case 10:
			w.functionSizes, err = readCodeFunctionSizes(sectionReader)
		default:
		}
		if err != nil {
			return fmt.Errorf("read WebAssembly section %q: %w", name, err)
		}
		if _, err = io.Copy(io.Discard, sectionReader); err != nil {
			return fmt.Errorf("read WebAssembly section %q: %w", name, err)
		}
		if limited.N != 0 {
			return fmt.Errorf("WebAssembly section %q is truncated", name)
		}

		offset += uint64(payloadSize)
		w.sections[name] = wasmSection{
			offset: sectionOffset,
			size:   offset - sectionOffset,
		}
	}
	return nil
}

func (*WasmWrapper) Text() (textStart uint64, text []byte, err error) {
	return textStart, nil, errors.New("text section not supported")
}

func (*WasmWrapper) GoArch() string {
	return "wasm"
}

func (w *WasmWrapper) ReadAddr(addr, size uint64) ([]byte, error) {
	end := addr + size
	if end > uint64(len(w.memory)) || end < addr {
		return nil, fmt.Errorf("read addr 0x%x size 0x%x out of range (memory size 0x%x)", addr, size, len(w.memory))
	}
	return w.memory[addr:end], nil
}

func (*WasmWrapper) LoadSymbols(func(name string, addr uint64, size uint64, typ entity.AddrType), func(addr uint64, size uint64)) error {
	return errors.New("load symbols not supported")
}

func (w *WasmWrapper) LoadSections() *entity.Store {
	store := entity.NewStore()

	// Linear memory is modeled as a virtual address space so that type descriptors
	// and pclntab metadata (which use linear-memory offsets as addresses) can be
	// looked up via IsData(), without contributing to file-size accounting.
	memSize := uint64(len(w.memory))
	if memSize > 0 {
		store.Sections["memory.data"] = &entity.Section{
			Name:           "memory.data",
			Size:           memSize,
			Addr:           0,
			AddrEnd:        memSize,
			VirtualSection: true,
			ContentType:    entity.SectionContentData,
		}
	}

	return store
}

func (*WasmWrapper) DWARF() (*dwarf.Data, error) {
	return nil, errors.New("dwarf section not supported")
}

// mergeIntervals merges a pre-sorted slice of [start, end) intervals,
// combining any that overlap or are adjacent.
func mergeIntervals(raw [][2]uint64) [][2]uint64 {
	merged := make([][2]uint64, 0, len(raw))
	for _, r := range raw {
		if len(merged) > 0 && r[0] <= merged[len(merged)-1][1] {
			if r[1] > merged[len(merged)-1][1] {
				merged[len(merged)-1][1] = r[1]
			}
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}

// dataSegmentRanges returns the sorted, merged virtual-address intervals
// [start, end) covered by active (non-passive) DataSegments. These are the
// ranges in linear memory that have actual file-backed bytes.
func (w *WasmWrapper) wasmDataSegmentRanges() [][2]uint64 {
	type interval = [2]uint64
	raw := make([]interval, 0, len(w.module.Data))
	for i := range w.module.Data {
		d := &w.module.Data[i]
		if d.Mode != wasmir.DataSegmentModeActive || d.MemoryIndex != 0 {
			continue
		}
		if len(d.OffsetExpr) == 0 || d.OffsetExpr[0].Kind != wasmir.InstrI32Const {
			continue // only i32.const offsets are used by Go-compiled Wasm
		}
		off := d.OffsetExpr[0].I32Const
		if off < 0 {
			continue
		}
		start := uint64(off)
		end := start + uint64(len(d.Init))
		if end > start {
			raw = append(raw, interval{start, end})
		}
	}
	slices.SortFunc(raw, func(a, b interval) int { return cmp.Compare(a[0], b[0]) })
	return mergeIntervals(raw)
}

func wasmMergedSymbolRanges(symbols entity.AddrSpace, typ entity.AddrType) [][2]uint64 {
	type interval = [2]uint64
	raw := make([]interval, 0, len(symbols))
	for _, addr := range symbols {
		if addr.Type != typ || addr.Size == 0 {
			continue
		}

		start := addr.Addr
		end := start + addr.Size
		if end < start {
			end = ^uint64(0)
		}

		raw = append(raw, interval{start, end})
	}

	slices.SortFunc(raw, func(a, b interval) int { return cmp.Compare(a[0], b[0]) })
	return mergeIntervals(raw)
}

// ComputeDataSectUsed returns the number of file-backed bytes in the Wasm
// data section covered by attributed data symbols. It intersects each symbol's
// virtual-address range with the actual DataSegment intervals so that
// zero-initialized linear-memory pages do not inflate the count.
func (w *WasmWrapper) ComputeDataSectUsed(symbols entity.AddrSpace) uint64 {
	segmentRanges := w.wasmDataSegmentRanges()
	if len(segmentRanges) == 0 {
		return 0
	}

	symbolRanges := wasmMergedSymbolRanges(symbols, entity.AddrTypeData)
	if len(symbolRanges) == 0 {
		return 0
	}

	total := uint64(0)
	symIdx := 0
	segIdx := 0

	for symIdx < len(symbolRanges) && segIdx < len(segmentRanges) {
		sym := symbolRanges[symIdx]
		seg := segmentRanges[segIdx]

		switch {
		case sym[1] <= seg[0]:
			symIdx++
		case seg[1] <= sym[0]:
			segIdx++
		default:
			lo := max(sym[0], seg[0])
			hi := min(sym[1], seg[1])
			if lo < hi {
				total += hi - lo
			}
			if sym[1] <= seg[1] {
				symIdx++
			} else {
				segIdx++
			}
		}
	}
	return total
}

func (w *WasmWrapper) GetSections(codeSectUsed, dataSectUsed uint64) []*entity.Section {
	ret := make([]*entity.Section, 0, len(w.sections))
	for name, sect := range w.sections {
		knownSize := uint64(0)
		isDebug := strings.HasPrefix(name, ".debug") || strings.HasPrefix(name, "custom_.debug")
		fileSize := sect.size
		if name == "code" {
			if codeSectUsed <= fileSize {
				knownSize = codeSectUsed
			} else {
				knownSize = fileSize
				slog.Warn("known code size is greater than code section size")
			}
		} else if name == "data" {
			if dataSectUsed <= fileSize {
				knownSize = dataSectUsed
			} else {
				knownSize = fileSize
				slog.Warn("known data size is greater than data section size")
			}
		}

		ret = append(ret, &entity.Section{
			Name:      name,
			Size:      fileSize,
			FileSize:  fileSize,
			KnownSize: knownSize,
			Offset:    sect.offset,
			End:       sect.offset + fileSize,
			Debug:     isDebug,
		})
	}
	return ret
}

var (
	infoStart, _ = hex.DecodeString("3077af0c9274080241e1c107e6d618e6")
	infoEnd, _   = hex.DecodeString("f932433186182072008242104116d8f2")
)

func (w *WasmWrapper) GetModInfo() *debug.BuildInfo {
	data := w.memory

	startMarkerLocation := bytes.Index(data, infoStart)
	if startMarkerLocation == -1 {
		return nil
	}

	searchForEndMarkerFrom := startMarkerLocation + len(infoStart)
	if searchForEndMarkerFrom > len(data) {
		return nil
	}

	remainingData := data[searchForEndMarkerFrom:]
	endMarkerRelativeLocation := bytes.Index(remainingData, infoEnd)

	if endMarkerRelativeLocation == -1 {
		return nil
	}

	sliceEndIndex := searchForEndMarkerFrom + endMarkerRelativeLocation + len(infoEnd)

	modinfo := string(data[startMarkerLocation:sliceEndIndex])

	bi, err := debug.ParseBuildInfo(modinfo)
	if err != nil {
		return nil
	}
	return bi
}

var _ RawFileWrapper = (*WasmWrapper)(nil)
