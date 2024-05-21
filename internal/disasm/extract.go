package disasm

import (
	"github.com/Zxilly/go-size-analyzer/internal/utils"

	"golang.org/x/arch/x86/x86asm"
)

var extractFuncs = map[string]extractorFunc{
	"amd64": extractAmd64,
}

func extractAmd64(code []byte, pc uint64) []PossibleStr {
	resultSet := utils.NewSet[PossibleStr]()

	var insts = make([]x86PosInst, 0)

	for len(code) > 0 {
		inst, err := x86asm.Decode(code, 64)
		size := inst.Len
		if err != nil || size == 0 || inst.Op == 0 {
			size = 1
		} else {
			if inst.Op != x86asm.NOP {
				insts = append(insts, x86PosInst{pc: pc, inst: inst})
			}
		}
		code = code[size:]
		pc += uint64(size)
	}

	for i := range len(insts) {
		for _, p := range x86Patterns {
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
