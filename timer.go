package ltask

import (
	"time"
	"unsafe"
)

const (
	Centisecond = time.Millisecond * 10
)

type timer struct {
	starttime    int64
	current      int64
	currentPoint int64
}

func newTimer() (t *timer) {
	ptr := malloc.Alloc(uint(unsafe.Sizeof(*t)))
	t = (*timer)(ptr)
	t.init()

	now := time.Now().UnixNano()
	t.starttime = now / int64(10*time.Millisecond)
	t.current = t.starttime % 100
	t.currentPoint = now / int64(10*time.Millisecond)

	return t
}

func (t *timer) init() {
	// TODO: Initialize timer
}

func (t *timer) destroy() {
	if t == nil {
		return
	}
	// TODO: Cleanup timer resources
	malloc.Free(unsafe.Pointer(t))
}
