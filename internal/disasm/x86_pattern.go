package disasm

import (
	"golang.org/x/arch/x86/x86asm"
)

type x86PosInst struct {
	pc   uint64
	inst x86asm.Inst
}

type x86Pattern struct {
	windowSize int
	matchFunc  func([]x86PosInst) *PossibleStr
}

func x86CountNotNilArgs(args x86asm.Args) int {
	cnt := 0
	for _, a := range args {
		if a != nil {
			cnt++
		}
	}
	return cnt
}

func x86GetMovImm(inst x86asm.Inst) (uint64, bool) {
	if x86CountNotNilArgs(inst.Args) != 2 {
		return 0, false
	}

	if inst.Op != x86asm.MOV {
		return 0, false
	}

	secondImm, ok := inst.Args[1].(x86asm.Imm)
	if !ok {
		return 0, false
	}
	if secondImm <= 0 {
		return 0, false
	}

	secondImmVal := uint64(secondImm)
	return secondImmVal, true
}

// currently only two x86Pattern get supported.
var x86Patterns = []x86Pattern{
	{
		windowSize: 2,
		// 1.
		//
		//	main.go:74            0x48066e                e80d24fbff              CALL runtime.printlock(SB)
		//	main.go:74            0x480673                488d05fde50100          LEAQ 0x1e5fd(IP), AX
		//	main.go:74            0x48067a                bb1e000000              MOVL $0x1e, BX
		//	main.go:74            0x48067f                90                      NOPL
		//	main.go:74            0x480680                e87b2cfbff              CALL runtime.printstring(SB)
		//	main.go:74            0x480685                e85624fbff              CALL runtime.printunlock(SB)
		matchFunc: func(insts []x86PosInst) *PossibleStr {
			first := insts[0]
			firstInst := first.inst
			if x86CountNotNilArgs(firstInst.Args) != 2 {
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
			imm, ok := x86GetMovImm(second.inst)
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
		// 2. todo: implement this, this is common for read-write data
		//
		//	main.go:79            0x4806ae                e8cd23fbff              CALL runtime.printlock(SB)
		//	main.go:79            0x4806b3                488b0536600a00          MOVQ main.GlobalString(SB), AX
		//	main.go:79            0x4806ba                488b1d37600a00          MOVQ main.GlobalString+8(SB), BX
		//	main.go:79            0x4806c1                e83a2cfbff              CALL runtime.printstring(SB)
		//	main.go:79            0x4806c6                e8f525fbff              CALL runtime.printnl(SB)
		//	main.go:79            0x4806cb                e81024fbff              CALL runtime.printunlock(SB)
		// matchFunc: func(insts []x86PosInst) *PossibleStr {
		// 	first := insts[0]
		// 	firstInst := first.inst
		// 	if firstInst.Op != x86asm.MOV {
		// 		return nil
		// 	}
		// 	if x86CountNotNilArgs(firstInst.Args) != 2 {
		// 		return nil
		// 	}

		// 	second := insts[1]
		// 	secondInst := second.inst
		// 	if secondInst.Op != x86asm.MOV {
		// 		return nil
		// 	}
		// 	if x86CountNotNilArgs(secondInst.Args) != 2 {
		// 		return nil
		// 	}

		// 	return nil
		// },
		matchFunc: func(_ []x86PosInst) *PossibleStr {
			return nil
		},
	},
}
