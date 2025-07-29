// +build amd64

#include "textflag.h"

// func unsafeString(ptr *byte, len int) string
TEXT ·unsafeString(SB), NOSPLIT, $0-32
	// string is two words: data pointer and length
	MOVQ	ptr+0(FP), AX      // AX = ptr
	MOVQ	len+8(FP), BX      // BX = len
	MOVQ	AX, ret+16(FP)     // string.data
	MOVQ	BX, ret+24(FP)     // string.len
	RET

