package disasm

import (
	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/arch/x86/x86asm"
)

var extractFuncs = map[string]extractorFunc{
	"amd64": extractAmd64,
}

// currently only two pattern get supported.
// 1.
//
//	main.go:74            0x48066e                e80d24fbff              CALL runtime.printlock(SB)
//	main.go:74            0x480673                488d05fde50100          LEAQ 0x1e5fd(IP), AX
//	main.go:74            0x48067a                bb1e000000              MOVL $0x1e, BX
//	main.go:74            0x48067f                90                      NOPL
//	main.go:74            0x480680                e87b2cfbff              CALL runtime.printstring(SB)
//	main.go:74            0x480685                e85624fbff              CALL runtime.printunlock(SB)
//
// 2.
//
//	main.go:79            0x4806ae                e8cd23fbff              CALL runtime.printlock(SB)
//	main.go:79            0x4806b3                488b0536600a00          MOVQ main.GlobalString(SB), AX
//	main.go:79            0x4806ba                488b1d37600a00          MOVQ main.GlobalString+8(SB), BX
//	main.go:79            0x4806c1                e83a2cfbff              CALL runtime.printstring(SB)
//	main.go:79            0x4806c6                e8f525fbff              CALL runtime.printnl(SB)
//	main.go:79            0x4806cb                e81024fbff              CALL runtime.printunlock(SB)
func extractAmd64(code []byte, pc uint64) []PossibleStr {
	type posInst struct {
		pc   uint64
		inst x86asm.Inst
	}

	type pattern struct {
		windowSize int
		matchFunc  func([]posInst) *PossibleStr
	}

	countNotNilArgsX86 := func(args x86asm.Args) int {
		cnt := 0
		for _, a := range args {
			if a != nil {
				cnt++
			}
		}
		return cnt
	}

	getMovImm := func(inst x86asm.Inst) (uint64, bool) {
		if countNotNilArgsX86(inst.Args) != 2 {
			return 0, false
		}

		if inst.Op != x86asm.MOV {
			return 0, false
		}

		secondImm, ok := inst.Args[1].(x86asm.Imm)
		if !ok {
			return 0, false
		}
		// this pattern can also be like set function parameters
		if secondImm <= 0 {
			return 0, false
		}

		secondImmVal := uint64(secondImm)
		return secondImmVal, true
	}

	patterns := []pattern{
		{
			windowSize: 2,
			matchFunc: func(insts []posInst) *PossibleStr {
				first := insts[0]
				firstInst := first.inst
				if countNotNilArgsX86(firstInst.Args) != 2 {
					return nil
				}
				if firstInst.Op != x86asm.LEA {
					return nil
				}
				firstMem, ok := firstInst.Args[1].(x86asm.Mem)
				if !ok {
					return nil
				}
				if firstMem.Base != x86asm.RIP {
					return nil
				}
				absAddr := first.pc + uint64(firstInst.Len) + uint64(firstMem.Disp)

				second := insts[1]
				imm, ok := getMovImm(second.inst)
				if !ok {
					return nil
				}

				return &PossibleStr{
					Addr: absAddr,
					Size: imm,
				}
			},
		},
		{
			windowSize: 2,
			matchFunc: func(insts []posInst) *PossibleStr {
				first := insts[0]
				firstInst := first.inst
				if countNotNilArgsX86(firstInst.Args) != 2 {
					return nil
				}
				if firstInst.Op != x86asm.LEA {
					return nil
				}
				firstMem, ok := firstInst.Args[1].(x86asm.Mem)
				if !ok {
					return nil
				}
				if firstMem.Base != x86asm.RIP {
					return nil
				}
				absAddr := first.pc + uint64(firstInst.Len) + uint64(firstMem.Disp)

				second := insts[1]
				imm, ok := getMovImm(second.inst)
				if !ok {
					return nil
				}

				return &PossibleStr{
					Addr: absAddr,
					Size: imm,
				}
			},
		},
	}

	resultSet := mapset.NewSet[PossibleStr]()

	var insts = make([]posInst, 0)

	for len(code) > 0 {
		inst, err := x86asm.Decode(code, 64)
		size := inst.Len
		if err != nil || size == 0 || inst.Op == 0 {
			size = 1
		} else {
			if inst.Op != x86asm.NOP {
				insts = append(insts, posInst{pc: pc, inst: inst})
			}
		}
		code = code[size:]
		pc += uint64(size)
	}

	for i := range insts {
		for _, p := range patterns {
			if len(insts) < i+p.windowSize {
				continue
			}
			matchRet := p.matchFunc(insts[i : i+p.windowSize])
			if matchRet != nil {
				resultSet.Add(*matchRet)
			}
		}
	}

	return resultSet.ToSlice()
}
