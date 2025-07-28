package util

/*
  see comment in v2/network/session.go about the usage
*/

//const (
//	SLIDE_BUFFER_SIZE = 1024 * 1024 * 4 // need to be tunable, possibly via urlOptions https://github.com/sijms/go-ora
//)

type SlideBufferHolder struct {
	head        int
	size        int
	ver         int
	initBufSize int
	buf         []byte
}

type SlideBuffer struct {
	hold *SlideBufferHolder
	ver  int
	// buf []byte // TODO? holds the current slice, may be available after the next SlideBufferHolder.NewSlideBuffer, updated every SlideBuffer.Write
}

func NewSlideBufferHolder(initBufSize int) *SlideBufferHolder {
	return &SlideBufferHolder{
		initBufSize: initBufSize,
		buf:         make([]byte, initBufSize),
	}
}

func (h *SlideBufferHolder) NewSlideBuffer( /*chunkSize int*/ ) *SlideBuffer {
	h.head += h.size
	//if h.head+chunkSize > SLIDE_BUFFER_SIZE {
	//	h.buf = make([]byte, SLIDE_BUFFER_SIZE)
	//	h.head = 0
	//}
	h.size = 0
	h.ver += 1
	return &SlideBuffer{
		hold: h,
		ver:  h.ver,
	}
}

func (h *SlideBufferHolder) ensureSize(l int) {
	if /*h.head+h.size+l > h.initBufSize &&*/ h.head+h.size+l > len(h.buf) {
		newBufSize := h.initBufSize
		if h.head == 0 || h.size+l > h.initBufSize {
			newBufSize = h.size + l + h.initBufSize/4
		}
		newBuf := make([]byte, newBufSize)
		//if h.size != 0 {
		copy(newBuf, h.buf[h.head:h.head+h.size])
		//}
		h.buf = newBuf
		h.head = 0
	}
}

// = SlideBufferHolder.NewSlideBuffer().Alloc(n).Bytes() but without SlideBuffer allocation and checks
func (h *SlideBufferHolder) AllocBytes(n int) []byte {
	h.head += h.size
	h.size = 0
	h.ver += 1
	h.ensureSize(n)
	h.size += n
	return h.buf[h.head : h.head+h.size]
}

// useless, (just for fun) Write-> copy(Alloc(len(p)).Bytes(), p), but just once, or see AllocBytes
// func (b *SlideBuffer) Alloc(n int) *SlideBuffer {
// 	h := b.hold
// 	if b.ver != h.ver {
// 		panic("ConcurrentModificationException")
// 	}
// 	h.ensureSize(n)
// 	h.size += n
// 	return b
// }

func (b *SlideBuffer) Write(p []byte) (n int, err error) {
	h := b.hold
	if b.ver != h.ver {
		panic("ConcurrentModificationException")
	}
	h.ensureSize(len(p))
	s := copy(h.buf[h.head+h.size:], p)
	h.size += s
	// b.buf = h.buf[h.head:h.head+h.size]
	return s, nil
}

func (b *SlideBuffer) Bytes() []byte {
	h := b.hold
	if b.ver != h.ver {
		panic("ConcurrentModificationException")
	}
	return h.buf[h.head : h.head+h.size]
	// return b.buf and comment out everything above
}
