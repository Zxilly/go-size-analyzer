package disasm

import (
	"bytes"
	"encoding/binary"

	"golang.org/x/arch/x86/x86asm"
)

type barrierCalls map[uint64]bool

// SetWriteBarrierCalls receives addresses resolved from runtime's pclntab.
// LoweredWB preserves program registers and writes only the GC buffer.
func (e *Extractor) SetWriteBarrierCalls(addresses []uint64) {
	calls := make(barrierCalls, len(addresses))
	for _, addr := range addresses {
		calls[addr] = true
	}
	switch e.goarch {
	case "amd64":
		e.extractor = func(code []byte, pc uint64) []PossibleStr { return extractAmd64WithBarriers(code, pc, calls) }
	case "arm64":
		e.extractor = func(code []byte, pc uint64) []PossibleStr { return extractArm64WithBarriers(code, pc, calls) }
	default:
	}
}

func (s *wordTracker) beginBarrier() {
	s.regs = [32]trackedWord{}
	for i := range s.stores {
		if !s.valid(s.stores[i].word) {
			s.stores[i].word = trackedWord{}
		}
	}
}

func (s *wordTracker) endBarrier(scratch int) {
	s.regs = [32]trackedWord{}
	for i := range s.stores {
		if s.stores[i].base == scratch {
			s.stores[i].word = trackedWord{}
		} else {
			s.stores[i].word.age = s.clock
		}
	}
}

func x86BarrierBlock(code []byte, pc uint64, calls barrierCalls) bool {
	if len(calls) == 0 || len(code) > 128 || !bytes.Contains(code, []byte{0xe8}) {
		return false
	}
	found := false
	for len(code) > 0 {
		inst, err := x86asm.Decode(code, 64)
		if err != nil || inst.Len == 0 {
			return false
		}
		switch inst.Op {
		case x86asm.NOP:
		case x86asm.CALL:
			rel, ok := inst.Args[0].(x86asm.Rel)
			if !ok || !calls[pc+uint64(inst.Len)+uint64(rel)] {
				return false
			}
			found = true
		case x86asm.MOV:
			if mem, ok := inst.Args[0].(x86asm.Mem); ok {
				if !found || mem.Base != x86asm.R11 || mem.Index != 0 {
					return false
				}
			}
		default:
			return false
		}
		code = code[inst.Len:]
		pc += uint64(inst.Len)
	}
	return found
}

func armBarrierBlock(code []byte, pc uint64, calls barrierCalls) bool {
	if len(calls) == 0 || len(code) > 128 || len(code)%4 != 0 {
		return false
	}
	found := false
	for len(code) > 0 {
		v := binary.LittleEndian.Uint32(code)
		switch {
		case v&0xfc000000 == 0x94000000:
			if !calls[pc+uint64(signExtend(v&0x3ffffff, 26)*4)] {
				return false
			}
			found = true
		case v&0xffc00000 == 0xf9000000:
			if !found || (v>>5)&31 != 25 {
				return false
			}
		case v&0xffc00000 == 0xf9400000:
		case v == 0xd503201f:
		default:
			return false
		}
		code = code[4:]
		pc += 4
	}
	return found
}
