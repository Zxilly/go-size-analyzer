package disasm

import (
	"golang.org/x/arch/x86/x86asm"

	"github.com/Zxilly/go-size-analyzer/internal/utils"
)

var extractFuncs = map[string]extractorFunc{
	"amd64": extractAmd64,
}

func extractAmd64(code []byte, pc uint64) []PossibleStr {
	resultSet := utils.NewSet[PossibleStr]()

	// Current patterns consume two non-NOP instructions. Keeping only that
	// window avoids allocating an instruction slice for every function.
	var window [2]x86PosInst
	count := 0

	for len(code) > 0 {
		inst, err := x86asm.Decode(code, 64)
		size := 0
		if err != nil || inst.Len == 0 || inst.Op == 0 {
			size = 1
		} else {
			size = inst.Len
			if inst.Op != x86asm.NOP {
				window[0] = window[1]
				window[1] = x86PosInst{pc: pc, inst: inst}
				count++
				if count >= len(window) {
					for _, p := range x86Patterns {
						if match := p.matchFunc(window[:]); match != nil {
							resultSet.Add(*match)
						}
					}
				}
			}
		}
		code = code[size:]
		pc += uint64(size)
	}

	return resultSet.ToSlice()
}
