// +build arm64

#include "textflag.h"

// func unsafeString(ptr *byte, len int) string
TEXT ·unsafeString(SB), NOSPLIT, $0-32
	// string is two words: data pointer and length
	MOVD	ptr+0(FP), R0      // R0 = ptr
	MOVD	len+8(FP), R1      // R1 = len
	MOVD	R0, ret+16(FP)     // string.data
	MOVD	R1, ret+24(FP)     // string.len
	RET
