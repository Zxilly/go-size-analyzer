package disasm

import "github.com/eliben/watgo/wasmir"

type wasmValue struct {
	word     trackedWord
	relative bool
	base     int
	offset   int64
}

// ExtractWasm follows statically known words and stack-relative addresses.
// Control boundaries discard state; dynamic program strings are not evaluated.
func ExtractWasm(body []wasmir.Instruction) []PossibleStr {
	s := newWordTracker()
	s.constantPointers = true
	var stack []wasmValue
	locals := map[uint32]wasmValue{}
	globals := map[uint32]wasmValue{}
	reset := func() { s.reset(); stack = nil; clear(locals); clear(globals) }
	pop := func() wasmValue {
		if len(stack) == 0 {
			return wasmValue{}
		}
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return v
	}
	push := func(v wasmValue) {
		if len(stack) >= 64 {
			reset()
			return
		}
		stack = append(stack, v)
	}
	constant := func(n uint64) wasmValue {
		return wasmValue{word: trackedWord{kind: wordConstant, value: n, age: s.clock}}
	}
	for _, inst := range body {
		s.clock++
		switch inst.Kind {
		case wasmir.InstrI32Const:
			push(constant(uint64(uint32(inst.I32Const))))
		case wasmir.InstrI64Const:
			push(constant(uint64(inst.I64Const)))
		case wasmir.InstrLocalGet:
			v, ok := locals[inst.LocalIndex]
			if !ok {
				v = wasmValue{relative: true, base: int(inst.LocalIndex) + 1}
			}
			push(v)
		case wasmir.InstrGlobalGet:
			v, ok := globals[inst.GlobalIndex]
			if !ok {
				v = wasmValue{relative: true, base: int(inst.GlobalIndex) + 1024}
			}
			push(v)
		case wasmir.InstrLocalSet, wasmir.InstrLocalTee:
			v := pop()
			if !v.relative && v.word.kind == wordUnknown {
				v = wasmValue{relative: true, base: int(s.clock) + 2048}
			}
			locals[inst.LocalIndex] = v
			if inst.Kind == wasmir.InstrLocalTee {
				push(v)
			}
		case wasmir.InstrGlobalSet:
			globals[inst.GlobalIndex] = pop()
		case wasmir.InstrI32WrapI64:
			v := pop()
			if !v.relative && v.word.kind == wordConstant {
				v.word.value = uint64(uint32(v.word.value))
			}
			push(v)
		case wasmir.InstrI64ExtendI32U, wasmir.InstrI64ExtendI32S:
			v := pop()
			if !v.relative && v.word.kind == wordConstant {
				if inst.Kind == wasmir.InstrI64ExtendI32S {
					v.word.value = uint64(int64(int32(v.word.value)))
				} else {
					v.word.value = uint64(uint32(v.word.value))
				}
			}
			push(v)
		case wasmir.InstrI32Add, wasmir.InstrI64Add, wasmir.InstrI32Sub, wasmir.InstrI64Sub:
			b, a := pop(), pop()
			sub := inst.Kind == wasmir.InstrI32Sub || inst.Kind == wasmir.InstrI64Sub
			if !sub && b.relative && !a.relative {
				a, b = b, a
			}
			if b.word.kind != wordConstant || b.relative {
				push(wasmValue{})
				continue
			}
			delta := b.word.value
			if sub {
				delta = 0 - delta
			}
			if a.relative {
				a.offset += int64(delta)
				push(a)
			} else if a.word.kind == wordConstant {
				n := a.word.value + delta
				if inst.Kind == wasmir.InstrI32Add || inst.Kind == wasmir.InstrI32Sub {
					n = uint64(uint32(n))
				}
				push(constant(n))
			} else {
				push(wasmValue{})
			}
		case wasmir.InstrI64Load:
			addr := pop()
			v := wasmValue{}
			if !addr.relative && addr.word.kind == wordConstant && inst.MemoryIndex == 0 {
				v.word = trackedWord{kind: wordLoad, value: addr.word.value + inst.MemoryOffset, age: s.clock}
			}
			push(v)
		case wasmir.InstrI64Store:
			value, addr := pop(), pop()
			if inst.MemoryIndex != 0 {
				reset()
				continue
			}
			if addr.relative {
				s.store(addr.base, addr.offset+int64(inst.MemoryOffset), value.word)
			} else if addr.word.kind == wordConstant {
				s.store(0, int64(addr.word.value+inst.MemoryOffset), value.word)
			} else {
				s.stores = [16]memoryWord{}
			}
		case wasmir.InstrMemoryCopy:
			length, source, _ := pop(), pop(), pop()
			if !source.relative && !length.relative && source.word.kind == wordConstant && length.word.kind == wordConstant && inst.MemoryIndex == 0 && inst.SourceMemoryIndex == 0 {
				s.result[PossibleStr{Addr: source.word.value, Size: length.word.value, Copy: true}] = struct{}{}
				// The compiler copies static aggregates with memory.copy. Inspect
				// pointer-aligned headers with a bounded amount of work per copy.
				for offset := uint64(0); offset+16 <= min(length.word.value, 64*1024); offset += 8 {
					s.result[PossibleStr{Addr: source.word.value + offset, Header: true}] = struct{}{}
				}
			}
			s.stores = [16]memoryWord{}
		case wasmir.InstrDrop:
			pop()
		case wasmir.InstrNop:
		default:
			reset()
		}
	}
	return s.candidates()
}
