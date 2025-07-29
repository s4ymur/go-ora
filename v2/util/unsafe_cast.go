package util

import (
	"unsafe"
)

var _PLACEHOLDERSTRVAL string = "PLACEHOLDERSTR"
var _PLACEHOLDERANYSTR any = _PLACEHOLDERSTRVAL
var _PLACEHOLDERSTRVALSZ int = int(unsafe.Sizeof(_PLACEHOLDERSTRVAL))
var _PLACEHOLDERUINTPTR uintptr = 0
var _PLACEHOLDERUINTPTRSZ int = int(unsafe.Sizeof(_PLACEHOLDERUINTPTR))

func _castStringToAnyStr(sbh *SlideBufferHolder, val string) (tmpany any) {
	// allocate memory for "string descriptor", 16 bytes
	var tmpslc []byte = sbh.AllocBytes(_PLACEHOLDERSTRVALSZ)
	// copy "string descriptor" to the created slice (see convTstring)
	*(*string)(unsafe.Pointer(&tmpslc[0])) = val
	// copy "string any" into ANY, 16 bytes, first 8 bytes of ANY is pointer to "string type"
	tmpany = _PLACEHOLDERANYSTR
	// the second 8 bytes of ANY is pointer to "string descriptor"
	*(*uintptr)(unsafe.Add(unsafe.Pointer(&tmpany), _PLACEHOLDERUINTPTRSZ)) = (uintptr)(unsafe.Pointer(&tmpslc[0]))
	return
}

// string -> any is in general: convTstring(string->any) (see comment below)
func CastStringToAnyStr(sbh *SlideBufferHolder, val string) any {
	if sbh == nil {
		return val
	} else if val == "" {
		return ""
	}
	return _castStringToAnyStr(sbh, val)
	// var tmpslc []byte = sbh.AllocBytes(_PLACEHOLDERSTRVALSZ)
	// *(*string)(unsafe.Pointer(&tmpslc[0])) = val
	// tmpany = _PLACEHOLDERANYSTR
	// *(*uintptr)(unsafe.Add(unsafe.Pointer(&tmpany), _PLACEHOLDERUINTPTRSZ)) = (uintptr)(unsafe.Pointer(&tmpslc[0]))
	// return
}

// Should be guaranteed that "val" slice is never reused (or allocated on stack); must come, for example from (another) SlideBuffer
// slice -> string -> any is in general:
// - slicebytetostring(byte[]->string) (mallocgc + memmove) - create memory region for "string slice" and copy data
// - convTstring(string->any) (mallocgc) - create memory region for "string descriptor"
// slicebytetostringtmp doesn't create memory region and doesn't copy data, but needs care
func CastSliceToAnyStr(sbh *SlideBufferHolder, val []byte) any {
	if sbh == nil {
		return string(val)
	} else if len(val) == 0 {
		return ""
	}
	return _castStringToAnyStr(sbh, slicebytetostringtmp(&val[0], len(val)))
	// var tmpslc []byte = sbh.AllocBytes(_PLACEHOLDERSTRVALSZ)
	// *(*string)(unsafe.Pointer(&tmpslc[0])) = slicebytetostringtmp(&val[0], len(val))
	// tmpany = _PLACEHOLDERANYSTR
	// *(*uintptr)(unsafe.Add(unsafe.Pointer(&tmpany), _PLACEHOLDERUINTPTRSZ)) = (uintptr)(unsafe.Pointer(&tmpslc[0]))
	// return
}

func slicebytetostringtmp(ptr *byte, n int) string {
	return unsafe.String(ptr, n)
}
