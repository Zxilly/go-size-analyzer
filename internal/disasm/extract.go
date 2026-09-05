package disasm

import "golang.org/x/arch/x86/x86asm"

var (
	extractFuncs = map[string]extractorFunc{"amd64": extractAmd64, "arm64": extractArm64}
	amd64ABI     = [...]int{0, 3, 1, 7, 6, 8, 9, 10, 11}
)

func x86Register(arg x86asm.Arg) int {
	r, ok := arg.(x86asm.Reg)
	if !ok {
		return -1
	}
	switch {
	case r >= x86asm.RAX && r <= x86asm.R15:
		return int(r - x86asm.RAX)
	case r >= x86asm.EAX && r <= x86asm.R15L:
		return int(r - x86asm.EAX)
	default:
		return -1
	}
}

func ripAddress(mem x86asm.Mem, pc uint64, n int) (uint64, bool) {
	if mem.Base != x86asm.RIP || mem.Index != 0 {
		return 0, false
	}
	return pc + uint64(n) + uint64(int64(int32(mem.Disp))), true
}

func extractAmd64(code []byte, pc uint64) []PossibleStr {
	return extractAmd64WithBarriers(code, pc, nil)
}

func extractAmd64WithBarriers(code []byte, pc uint64, calls barrierCalls) []PossibleStr {
	s := newWordTracker()
	var barrierEnd uint64
	for len(code) > 0 {
		s.clock++
		if pc == barrierEnd {
			s.endBarrier(11)
			barrierEnd = 0
		}
		inst, err := x86asm.Decode(code, 64)
		if err != nil || inst.Len == 0 || inst.Op == 0 {
			s.reset()
			code = code[1:]
			pc++
			continue
		}
		dst := x86Register(inst.Args[0])
		w := trackedWord{}
		switch inst.Op {
		case x86asm.NOP, x86asm.CMP, x86asm.TEST:
		case x86asm.CALL:
			rel, ok := inst.Args[0].(x86asm.Rel)
			if barrierEnd > pc && ok && calls[pc+uint64(inst.Len)+uint64(rel)] {
				s.regs = [32]trackedWord{}
			} else {
				s.reset()
			}
		case x86asm.JE, x86asm.JNE:
			rel, ok := inst.Args[0].(x86asm.Rel)
			if ok && rel > 0 && int64(rel) <= int64(len(code)-inst.Len) && x86BarrierBlock(code[inst.Len:inst.Len+int(rel)], pc+uint64(inst.Len), calls) {
				s.beginBarrier()
				barrierEnd = pc + uint64(inst.Len) + uint64(rel)
			} else {
				s.reset()
			}
		case x86asm.MOVUPS, x86asm.MOVUPD, x86asm.MOVDQU, x86asm.MOVDQA, x86asm.MOVAPS:
			// Static aggregate copies can carry an entire string header in XMM.
			if mem, ok := inst.Args[1].(x86asm.Mem); ok && mem.Index == 0 {
				addr, known := ripAddress(mem, pc, inst.Len)
				if !known {
					base := s.get(x86Register(mem.Base))
					known = base.kind == wordAddress
					addr = base.value + uint64(mem.Disp)
				}
				if known {
					s.result[PossibleStr{Addr: addr, Header: true}] = struct{}{}
				}
			}
		case x86asm.LEA:
			if mem, ok := inst.Args[1].(x86asm.Mem); ok {
				if addr, ok := ripAddress(mem, pc, inst.Len); ok {
					w = trackedWord{kind: wordAddress, value: addr}
				}
			}
			s.set(dst, w)
			s.registers(amd64ABI[:])
		case x86asm.MOV:
			switch src := inst.Args[1].(type) {
			case x86asm.Imm:
				w = trackedWord{kind: wordConstant, value: uint64(src)}
			case x86asm.Reg:
				w = s.get(x86Register(src))
			case x86asm.Mem:
				if addr, ok := ripAddress(src, pc, inst.Len); ok && inst.DataSize == 64 {
					w = trackedWord{kind: wordLoad, value: addr}
				}
			default:
			}
			if dst >= 0 {
				if inst.DataSize == 32 {
					if w.kind == wordLoad {
						w = trackedWord{}
					} else {
						w.value = uint64(uint32(w.value))
					}
				}
				s.set(dst, w)
				s.registers(amd64ABI[:])
			} else if mem, ok := inst.Args[0].(x86asm.Mem); ok && inst.DataSize == 64 && mem.Index == 0 {
				if !(barrierEnd > pc && mem.Base == x86asm.R11) {
					s.store(x86Register(mem.Base), mem.Disp, w)
				}
			} else if _, ok := inst.Args[0].(x86asm.Reg); ok {
				s.reset()
			}
		case x86asm.ADD, x86asm.SUB, x86asm.AND, x86asm.OR, x86asm.XOR, x86asm.SHL, x86asm.SHR, x86asm.SAR, x86asm.INC, x86asm.DEC:
			// Conditional branches, calls and instructions with implicit writes form
			// an evidence boundary. Known arithmetic writes invalidate their target.
			if dst >= 0 {
				s.set(dst, trackedWord{})
			} else {
				s.reset()
			}
		default:
			s.reset()
		}
		code = code[inst.Len:]
		pc += uint64(inst.Len)
	}
	return s.candidates()
}
