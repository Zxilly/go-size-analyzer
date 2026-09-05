package wrapper

import (
	"bufio"
	"cmp"
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
	if len(w.fileMappings) == 0 || w.fileMappings[len(w.fileMappings)-1].Addr+w.fileMappings[len(w.fileMappings)-1].Size <= m.Addr {
		w.fileMappings = append(w.fileMappings, m)
		return
	}
	// Replace only the intersecting interval. Rebuilding and sorting the full
	// slice per segment allocates gigabytes for Go's sparse WASM data sections.
	lo, _ := slices.BinarySearchFunc(w.fileMappings, m.Addr, func(old entity.FileMapping, addr uint64) int {
		if old.Addr+old.Size <= addr {
			return -1
		}
		return 1
	})
	end := m.Addr + m.Size
	hi, _ := slices.BinarySearchFunc(w.fileMappings, end, func(old entity.FileMapping, addr uint64) int {
		return cmp.Compare(old.Addr, addr)
	})
	var replacement [3]entity.FileMapping
	parts := replacement[:0]
	if lo < hi && w.fileMappings[lo].Addr < m.Addr {
		left := w.fileMappings[lo]
		left.Size = m.Addr - left.Addr
		parts = append(parts, left)
	}
	parts = append(parts, m)
	if lo < hi && w.fileMappings[hi-1].Addr+w.fileMappings[hi-1].Size > end {
		right := w.fileMappings[hi-1]
		right.Size -= end - right.Addr
		right.Offset += end - right.Addr
		right.Addr = end
		parts = append(parts, right)
	}
	w.fileMappings = slices.Replace(w.fileMappings, lo, hi, parts...)
}
func (w *WasmWrapper) FileAddressMappings() []entity.FileMapping { return w.fileMappings }

func (w *WasmWrapper) FileDataContains(addr, size uint64) bool {
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
