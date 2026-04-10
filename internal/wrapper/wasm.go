package wrapper

import (
	"bytes"
	"cmp"
	"debug/dwarf"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/ZxillyFork/wazero/notinternal/leb128"
	"github.com/ZxillyFork/wazero/notinternal/wasm"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

type WasmWrapper struct {
	module *wasm.Module
	memory []byte
}

var _ RawFileWrapper = (*WasmWrapper)(nil)

const funcValueOffset = 0x1000

func (w *WasmWrapper) GetFunctionSize(idx uint64, meq125 bool) uint64 {
	// Go 1.25+ stores PC_F directly in pclntab, older versions store full PC (PC_F << 16)
	if !meq125 {
		idx = idx >> 16
	}
	idx = idx - funcValueOffset

	return uint64(len(w.module.CodeSection[idx].Body))
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
	raw := make([]interval, 0, len(w.module.DataSection))
	for i := range w.module.DataSection {
		d := &w.module.DataSection[i]
		if d.IsPassive() {
			continue
		}
		if d.OffsetExpression.Opcode != wasm.OpcodeI32Const {
			continue // only i32.const offsets are used by Go-compiled Wasm
		}
		off, _, err := leb128.LoadInt32(d.OffsetExpression.Data)
		if err != nil || off < 0 {
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
	ret := make([]*entity.Section, 0)
	for name, sect := range w.module.Sections {
		knownSize := uint64(0)
		isDebug := strings.HasPrefix(name, "custom_.debug")
		fileSize := uint64(sect.Size)
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
		} else if isDebug {
			knownSize = fileSize
		}

		ret = append(ret, &entity.Section{
			Name:      name,
			Size:      fileSize,
			FileSize:  fileSize,
			KnownSize: knownSize,
			Offset:    uint64(sect.Offset),
			End:       uint64(sect.Offset) + fileSize,
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
