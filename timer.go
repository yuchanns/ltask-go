package ltask

import "time"

type timer struct {
	starttime    int64
	current      int64
	currentPoint int64
}

func newTimer() *timer {
	t := &timer{}
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
