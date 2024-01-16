package objfile

import "golang.org/x/arch/x86/x86asm"

func countNotArgsX86(args x86asm.Args) int {
	cnt := 0
	for _, a := range args {
		if a != nil {
			cnt++
		}
	}
	return cnt
}
