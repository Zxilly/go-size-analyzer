package disasm

import (
	"encoding/binary"

	"golang.org/x/arch/arm64/arm64asm"
)

var arm64ABI = [...]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

func signExtend(v uint32, bits uint) int64 { return int64(int32(v<<(32-bits)) >> (32 - bits)) }
func armConstant(arg arm64asm.Arg) (uint64, bool) {
	switch v := arg.(type) {
	case arm64asm.Imm:
		return uint64(v.Imm), true
	case arm64asm.Imm64:
		return v.Imm, true
	}
	return 0, false
}

func extractArm64(code []byte, pc uint64) []PossibleStr {
	return extractArm64WithBarriers(code, pc, nil)
}

func extractArm64WithBarriers(code []byte, pc uint64, calls barrierCalls) []PossibleStr {
	s := newWordTracker()
	var barrierEnd uint64
	for len(code) >= 4 {
		s.clock++
		if pc == barrierEnd {
			s.endBarrier(25)
			barrierEnd = 0
		}
		bits := binary.LittleEndian.Uint32(code)
		rd, rn := int(bits&31), int((bits>>5)&31)
		op, err := arm64asm.Decode(code[:4])
		if err != nil {
			s.reset()
			code = code[4:]
			pc += 4
			continue
		}
		switch {
		case bits&0x7f000000 == 0x34000000: // CBZ skipping a bounded write-barrier block
			distance := signExtend((bits>>5)&0x7ffff, 19) * 4
			if distance > 4 && distance <= int64(len(code)) && armBarrierBlock(code[4:distance], pc+4, calls) {
				s.beginBarrier()
				barrierEnd = pc + uint64(distance)
			} else {
				s.reset()
			}
		case bits&0xfc000000 == 0x94000000:
			target := pc + uint64(signExtend(bits&0x3ffffff, 26)*4)
			if barrierEnd > pc && calls[target] {
				s.regs = [32]trackedWord{}
			} else {
				s.reset()
			}
		case bits&0x9f000000 == 0x90000000: // ADRP
			offset := signExtend(((bits>>5)&0x7ffff)<<2|((bits>>29)&3), 21) << 12
			s.set(rd, trackedWord{kind: wordAddress, value: (pc &^ 0xfff) + uint64(offset)})
		case bits&0x9f000000 == 0x10000000: // ADR
			offset := signExtend(((bits>>5)&0x7ffff)<<2|((bits>>29)&3), 21)
			s.set(rd, trackedWord{kind: wordAddress, value: pc + uint64(offset)})
		case bits&0xff000000 == 0x91000000: // ADD Xd, Xn, #imm
			w := s.get(rn)
			if w.kind == wordAddress || w.kind == wordConstant {
				w.value += uint64((bits>>10)&0xfff) << (((bits >> 22) & 1) * 12)
			} else {
				w = trackedWord{}
			}
			s.set(rd, w)
		case bits&0x7f800000 == 0x52800000: // MOVZ
			s.set(rd, trackedWord{kind: wordConstant, value: uint64((bits>>5)&0xffff) << (((bits >> 21) & 3) * 16)})
		case bits&0x7f800000 == 0x72800000: // MOVK
			w := s.get(rd)
			if w.kind == wordConstant {
				shift := ((bits >> 21) & 3) * 16
				w.value = w.value&^(uint64(0xffff)<<shift) | uint64((bits>>5)&0xffff)<<shift
			} else {
				w = trackedWord{}
			}
			s.set(rd, w)
		case bits&0xffc00000 == 0xf9400000: // LDR Xt, [Xn,#imm]
			w := s.get(rn)
			if w.kind == wordAddress {
				w = trackedWord{kind: wordLoad, value: w.value + uint64((bits>>10)&0xfff)*8}
			} else {
				w = trackedWord{}
			}
			s.set(rd, w)
		case bits&0xffc00000 == 0xb9400000: // LDR Wt does not alter stored object fields.
			s.set(rd, trackedWord{})
		case bits&0xffc00000 == 0xa9400000: // LDP Xt1, Xt2, [Xn,#imm]
			base := s.get(rn)
			w := trackedWord{}
			if base.kind == wordAddress {
				w = trackedWord{kind: wordLoad, value: base.value + uint64(signExtend((bits>>15)&127, 7)*8)}
			}
			s.set(rd, w)
			if w.kind != wordUnknown {
				w.value += 8
			}
			s.set(int((bits>>10)&31), w)
		case bits&0xffc00000 == 0xad400000: // LDP Qt1, Qt2: static aggregate copy
			base := s.get(rn)
			if base.kind == wordAddress {
				addr := base.value + uint64(signExtend((bits>>15)&127, 7)*16)
				s.result[PossibleStr{Addr: addr, Header: true}] = struct{}{}
				s.result[PossibleStr{Addr: addr + 16, Header: true}] = struct{}{}
			}
		case bits&0xffc00000 == 0xf9000000: // STR Xt, [Xn,#imm]
			if !(barrierEnd > pc && rn == 25) {
				s.store(rn, int64((bits>>10)&0xfff)*8, s.get(rd))
			}
		case bits&0xffc00000 == 0xa9000000: // STP Xt1, Xt2, [Xn,#imm]
			offset := signExtend((bits>>15)&127, 7) * 8
			s.store(rn, offset, s.get(rd))
			s.store(rn, offset+8, s.get(int((bits>>10)&31)))
		case op.Op == arm64asm.MOV || op.Op == arm64asm.ORR:
			if op.Op == arm64asm.ORR {
				r, ok := op.Args[1].(arm64asm.Reg)
				if !ok || (r != arm64asm.XZR && r != arm64asm.WZR) {
					s.set(rd, trackedWord{})
					break
				}
			}
			w := trackedWord{}
			for _, arg := range op.Args[1:] {
				if n, ok := armConstant(arg); ok {
					w = trackedWord{kind: wordConstant, value: n}
					break
				}
			}
			if w.kind == wordUnknown {
				if r, ok := op.Args[1].(arm64asm.Reg); ok && r >= arm64asm.X0 && r < arm64asm.XZR {
					w = s.get(int(r - arm64asm.X0))
				}
			}
			s.set(rd, w)
		case op.Op == arm64asm.NOP || op.Op == arm64asm.CMP || op.Op == arm64asm.TST:
		case bits&0x7c000000 == 0x14000000 || bits&0xff000010 == 0x54000000 || bits&0x7e000000 == 0x34000000 || bits&0x7e000000 == 0x36000000 || bits&0xfe000000 == 0xd6000000:
			s.reset()
		default:
			s.reset()
		}
		s.registers(arm64ABI[:])
		code = code[4:]
		pc += 4
	}
	return s.candidates()
}
