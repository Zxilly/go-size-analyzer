package wrapper

import (
	"bufio"
	"cmp"
	"container/heap"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/eliben/watgo/wasmir"

	"github.com/Zxilly/go-size-analyzer/internal/entity"
)

func (w *WasmWrapper) readCodeLayout(r *bufio.Reader, offset uint64) ([]uint64, []entity.FileRange, error) {
	count, n, err := readWasmUint32(r)
	if err != nil {
		return nil, nil, err
	}
	w.fileMetadata = append(w.fileMetadata, entity.FileRange{Offset: offset, Size: n})
	offset += n
	sizes := make([]uint64, 0, min(count, 1024))
	ranges := make([]entity.FileRange, 0, min(count, 1024))
	for range count {
		start := offset
		body, n, err := readWasmUint32(r)
		if err != nil {
			return nil, nil, err
		}
		offset += n
		locals, n, err := readWasmUint32(r)
		if err != nil {
			return nil, nil, err
		}
		localBytes := n
		for range locals {
			_, n, err := readWasmUint32(r)
			if err != nil {
				return nil, nil, err
			}
			localBytes += n
			if _, err = r.ReadByte(); err != nil {
				return nil, nil, err
			}
			localBytes++
		}
		if localBytes > uint64(body) {
			return nil, nil, errors.New("WebAssembly function locals exceed body size")
		}
		offset += localBytes
		size := uint64(body) - localBytes
		w.fileMetadata = append(w.fileMetadata, entity.FileRange{Offset: start, Size: offset - start})
		ranges = append(ranges, entity.FileRange{Offset: offset, Size: size})
		sizes = append(sizes, size)
		if _, err = io.CopyN(io.Discard, r, int64(size)); err != nil {
			return nil, nil, err
		}
		offset += size
	}
	return sizes, ranges, nil
}

func readWasmSigned(r io.ByteReader, bits uint) (int64, uint64, error) {
	var value uint64
	for i := uint(0); i < (bits+6)/7; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, uint64(i), err
		}
		value |= uint64(b&127) << (7 * i)
		if b&128 == 0 {
			shift := 7 * (i + 1)
			if b&64 != 0 && shift < 64 {
				value |= ^uint64(0) << shift
			}
			return int64(value), uint64(i + 1), nil
		}
	}
	return 0, 0, errors.New("WebAssembly signed integer overflow")
}

func (w *WasmWrapper) readDataLayout(r *bufio.Reader, offset uint64) error {
	count, n, err := readWasmUint32(r)
	if err != nil {
		return err
	}
	w.fileMetadata = append(w.fileMetadata, entity.FileRange{Offset: offset, Size: n})
	offset += n
	for range count {
		start := offset
		flags, n, err := readWasmUint32(r)
		if err != nil {
			return err
		}
		offset += n
		memory := uint32(0)
		known := flags != 1
		var addr int64
		if flags > 2 {
			return fmt.Errorf("unsupported data segment flags %d", flags)
		}
		if flags == 2 {
			memory, n, err = readWasmUint32(r)
			if err != nil {
				return err
			}
			offset += n
		}
		if flags != 1 {
			op, err := r.ReadByte()
			if err != nil {
				return err
			}
			offset++
			switch op {
			case 0x41:
				addr, n, err = readWasmSigned(r, 32)
			case 0x42:
				addr, n, err = readWasmSigned(r, 64)
			case 0x23:
				_, n, err = readWasmUint32(r)
				known = false
			default:
				return fmt.Errorf("unsupported data offset opcode %#x", op)
			}
			if err != nil {
				return err
			}
			offset += n
			end, err := r.ReadByte()
			if err != nil {
				return err
			}
			offset++
			if end != 0x0b {
				return errors.New("unsupported data offset expression")
			}
		}
		size, n, err := readWasmUint32(r)
		if err != nil {
			return err
		}
		offset += n
		w.fileMetadata = append(w.fileMetadata, entity.FileRange{Offset: start, Size: offset - start})
		if known && memory == 0 && addr >= 0 {
			w.addFileMapping(entity.FileMapping{Addr: uint64(addr), FileRange: entity.FileRange{Offset: offset, Size: uint64(size)}})
		}
		if _, err = io.CopyN(io.Discard, r, int64(size)); err != nil {
			return err
		}
		offset += uint64(size)
	}
	return nil
}

func (w *WasmWrapper) addFileMapping(m entity.FileMapping) {
	if m.Size == 0 {
		return
	}
	if m.Size > ^uint64(0)-m.Addr {
		return
	}
	if len(w.fileMappings) > 0 {
		last := w.fileMappings[len(w.fileMappings)-1]
		w.mappingsDirty = w.mappingsDirty || last.Addr+last.Size > m.Addr
	}
	w.fileMappings = append(w.fileMappings, m)
}

// Normalize in one sweep. The highest original segment index wins each
// interval, matching WebAssembly's later-initialization overwrite semantics.
// Every segment contributes two endpoints and at most one heap push/pop.
func (w *WasmWrapper) normalizeFileMappings() {
	if !w.mappingsDirty {
		return
	}
	type endpoint struct {
		addr  uint64
		index int
		start bool
	}
	events := make([]endpoint, 0, 2*len(w.fileMappings))
	for i, m := range w.fileMappings {
		events = append(events, endpoint{m.Addr, i, true}, endpoint{m.Addr + m.Size, i, false})
	}
	slices.SortFunc(events, func(a, b endpoint) int { return cmp.Compare(a.addr, b.addr) })
	active := make([]bool, len(w.fileMappings))
	var winners mappingHeap
	out := make([]entity.FileMapping, 0, len(w.fileMappings))
	for i := 0; i < len(events); {
		addr := events[i].addr
		for i < len(events) && events[i].addr == addr {
			e := events[i]
			active[e.index] = e.start
			if e.start {
				heap.Push(&winners, e.index)
			}
			i++
		}
		for len(winners) > 0 && !active[winners[0]] {
			heap.Pop(&winners)
		}
		if i == len(events) || len(winners) == 0 {
			continue
		}
		m := w.fileMappings[winners[0]]
		m.Offset += addr - m.Addr
		m.Addr, m.Size = addr, events[i].addr-addr
		if len(out) > 0 {
			last := &out[len(out)-1]
			if last.Addr+last.Size == m.Addr && last.Offset+last.Size == m.Offset {
				last.Size += m.Size
				continue
			}
		}
		out = append(out, m)
	}
	w.fileMappings, w.mappingsDirty = out, false
}

type mappingHeap []int

func (h mappingHeap) Len() int           { return len(h) }
func (h mappingHeap) Less(i, j int) bool { return h[i] > h[j] }
func (h mappingHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *mappingHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *mappingHeap) Pop() any          { old := *h; x := old[len(old)-1]; *h = old[:len(old)-1]; return x }

func (w *WasmWrapper) FileAddressMappings() []entity.FileMapping {
	w.normalizeFileMappings()
	return w.fileMappings
}

func (w *WasmWrapper) FileDataContains(addr, size uint64) bool {
	w.normalizeFileMappings()
	if size == 0 || addr > uint64(len(w.memory)) || size > uint64(len(w.memory))-addr {
		return false
	}
	i, _ := slices.BinarySearchFunc(w.fileMappings, addr, func(m entity.FileMapping, a uint64) int { return cmp.Compare(m.Addr, a) })
	for _, index := range []int{i, i - 1} {
		if index >= 0 && index < len(w.fileMappings) {
			m := w.fileMappings[index]
			if addr >= m.Addr && addr-m.Addr < m.Size {
				return true
			}
		}
	}
	return false
}
func (w *WasmWrapper) FileMetadataRanges() []entity.FileRange { return w.fileMetadata }

func (w *WasmWrapper) FileDataSize(coverage entity.AddrCoverage) uint64 {
	return w.mappedDataSize(len(coverage), func(i int) entity.AddrPos { return *coverage[i].Pos })
}

func (w *WasmWrapper) FileDataIntervals(ranges []entity.AddrPos) uint64 {
	return w.mappedDataSize(len(ranges), func(i int) entity.AddrPos { return ranges[i] })
}

func (w *WasmWrapper) mappedDataSize(count int, at func(int) entity.AddrPos) uint64 {
	w.normalizeFileMappings()
	var size uint64
	i, j := 0, 0
	if count > 0 {
		j, _ = slices.BinarySearchFunc(w.fileMappings, at(0).Addr, func(m entity.FileMapping, addr uint64) int {
			if m.Addr+m.Size <= addr {
				return -1
			}
			return 1
		})
	}
	for i < count && j < len(w.fileMappings) {
		r, m := at(i), w.fileMappings[j]
		end, mend := r.Addr+r.Size, m.Addr+m.Size
		if end <= m.Addr {
			i++
			continue
		}
		if mend <= r.Addr {
			advance, _ := slices.BinarySearchFunc(w.fileMappings[j+1:], r.Addr, func(m entity.FileMapping, addr uint64) int {
				if m.Addr+m.Size <= addr {
					return -1
				}
				return 1
			})
			j += advance + 1
			continue
		}
		size += min(end, mend) - max(r.Addr, m.Addr)
		if end <= mend {
			i++
		} else {
			j++
		}
	}
	return size
}

func (w *WasmWrapper) FunctionFileRange(pc uint64, modern bool) (entity.FileRange, bool) {
	if !modern {
		pc >>= 16
	}
	if pc < funcValueOffset {
		return entity.FileRange{}, false
	}
	index := pc - funcValueOffset
	if index >= uint64(len(w.functionRanges)) {
		return entity.FileRange{}, false
	}
	return w.functionRanges[index], true
}

func (w *WasmWrapper) RecognizedSection(name string) bool {
	s, ok := w.sections[name]
	if !ok {
		return false
	}
	if s.kind > 0 {
		return s.kind != 10 && s.kind != 11
	}
	switch s.originalName {
	case "name", "producers", "target_features", "go:buildid", "sourceMappingURL":
		return true
	}
	return false
}

func (w *WasmWrapper) FunctionInstructions(pc uint64, modern bool) ([]wasmir.Instruction, bool) {
	if !modern {
		pc >>= 16
	}
	if pc < funcValueOffset || w.module == nil {
		return nil, false
	}
	i := pc - funcValueOffset
	if i >= uint64(len(w.module.Funcs)) {
		return nil, false
	}
	return w.module.Funcs[i].Body, true
}
