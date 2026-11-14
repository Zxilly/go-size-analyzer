#include "textflag.h"
DATA cfgPayload<>+0(SB)/8, $"cfg_bloc"
DATA cfgPayload<>+8(SB)/8, $"k_string"
GLOBL cfgPayload<>(SB), RODATA, $16
TEXT ·cross(SB), NOSPLIT, $0-0
	LEAQ cfgPayload<>(SB), AX
	TESTQ CX, CX
	JE joined
	NOP
joined:
	MOVQ $16, BX
	RET
