package ltask

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/aristanetworks/goarista/monotime"
	"github.com/phuslu/log"
)

const (
	Centisecond = time.Millisecond * 10

	timeNearShift  = 8
	timeNear       = 1 << timeNearShift
	timeLevelShift = 6
	timeLevel      = 1 << timeLevelShift
	timeNearMask   = timeNear - 1
	timeLevelMask  = timeLevel - 1
)

type timerNode[T any] struct {
	next   *timerNode[T]
	expire int32
	event  T
}

type linkList[T any] struct {
	head timerNode[T]
	tail *timerNode[T]
}

func (l *linkList[T]) link(node *timerNode[T]) {
	l.tail.next = node
	l.tail = node
	node.next = nil
}

func (l *linkList[T]) clear() (ret *timerNode[T]) {
	ret = l.head.next
	l.head.next = nil
	l.tail = &l.head
	return
}

type timer[T any] struct {
	n            [timeNear]linkList[T]
	t            [4][timeLevel]linkList[T]
	l            int32
	time         uint32
	starttime    int64
	current      int64
	currentPoint uint64
}

func newTimer[T any]() (t *timer[T]) {
	ptr := malloc.Alloc(uint(unsafe.Sizeof(*t)))
	t = (*timer[T])(ptr)

	for i := range timeNear {
		t.n[i].clear()
	}

	for i := range 4 {
		for j := range timeLevel {
			t.t[i][j].clear()
		}
	}

	t.init()

	return t
}

func (t *timer[T]) assert() {
	if t == nil {
		panic("timer is nil")
	}
}

func (t *timer[T]) acquireLock() {
	t.assert()
	for !atomic.CompareAndSwapInt32(&t.l, 0, 1) {
		log.Debug().Msgf("timer acquireLock failed, waiting...%d", t.l)
		runtime.Gosched()
	}
}

func (t *timer[T]) releaseLock() {
	t.assert()
	atomic.StoreInt32(&t.l, 0)
}

func (t *timer[T]) start() int64 {
	t.assert()
	return t.starttime
}

func (t *timer[T]) now() int64 {
	t.assert()
	return t.current
}

func (t *timer[T]) init() {
	t.assert()
	now := time.Now()
	csec := now.UnixMilli() / 10
	t.time = 0
	t.starttime = csec / 100
	t.current = csec % 100
	t.currentPoint = uint64(time.Unix(0, int64(monotime.Now())).UnixMilli() / 10)
	atomic.StoreInt32(&t.l, 0)
}

func (t *timer[T]) add(event *T, time int32) {
	t.acquireLock()
	defer t.releaseLock()

	var node *timerNode[T]
	ptr := malloc.Alloc(uint(unsafe.Sizeof(*node)))
	node = (*timerNode[T])(ptr)
	node.next = nil
	node.event = *event
	node.expire = time + int32(t.time)
	t.addNode(node)
}

type timerExecuteFunc[T any] func(arg *T)

func (t *timer[T]) update(fn timerExecuteFunc[T]) {
	t.assert()
	cp := uint64(time.Unix(0, int64(monotime.Now())).UnixMilli() / 10)
	if cp < t.currentPoint {
		fmt.Printf("timer diff error: change from %d to %d\n", cp, t.currentPoint)
		t.currentPoint = cp
	} else if cp == t.currentPoint {
		return
	}
	diff := int64(cp - t.currentPoint)
	t.currentPoint = cp
	t.current += diff
	for i := int64(0); i < diff; i++ {
		t.tick(fn)
	}
}

func (t *timer[T]) tick(fn timerExecuteFunc[T]) {
	t.acquireLock()
	defer t.releaseLock()

	// try to dispatch timeout 0 (rare condition)
	t.execute(fn)

	// shift time first, and then dispatch timer message
	t.shift()

	t.execute(fn)
}

func (t *timer[T]) dispatchList(current *timerNode[T], fn timerExecuteFunc[T]) {
	for {
		if fn != nil {
			fn(&current.event)
		}
		temp := current
		current = current.next
		malloc.Free(unsafe.Pointer(temp))
		if current == nil {
			return
		}
	}
}

func (t *timer[T]) execute(fn timerExecuteFunc[T]) {
	t.assert()
	idx := t.time & timeNearMask

	for t.n[idx].head.next != nil {
		current := t.n[idx].clear()
		t.releaseLock()
		// dispatchList don't need lock
		t.dispatchList(current, fn)
		t.acquireLock()
	}
}

func (t *timer[T]) shift() {
	t.assert()
	mask := uint32(timeNear)
	t.time++
	ct := t.time
	if ct == 0 {
		t.moveList(3, 0)
		return
	}
	time := ct >> timeNearShift
	var i int
	for (ct & (mask - 1)) == 0 {
		idx := time & timeLevelMask
		if idx != 0 {
			t.moveList(i, int(idx))
			break
		}
		mask <<= timeLevelShift
		time >>= timeLevelShift
		i++
	}
}

func (t *timer[T]) moveList(level int, idx int) {
	t.assert()

	current := t.t[level][idx].clear()
	for current != nil {
		temp := current.next
		t.addNode(current)
		current = temp
	}
}

func (t *timer[T]) addNode(node *timerNode[T]) {
	t.assert()

	time := node.expire
	current_time := int32(t.time)

	if (time | timeNearMask) == (current_time | timeNearMask) {
		t.n[time&timeNearMask].link(node)
		return
	}
	mask := int32(timeNear << timeLevelShift)
	var i = 0
	for ; i < 3; i++ {
		if (time | (mask - 1)) == (current_time | (mask - 1)) {
			break
		}
		mask <<= timeLevelShift
	}
	t.t[i][(time>>(timeNearShift+i*timeLevelShift))&timeLevelMask].link(node)
}

func (t *timer[T]) destroy() {
	if t == nil {
		return
	}
	t.acquireLock()
	defer func() {
		t.releaseLock()
		malloc.Free(unsafe.Pointer(t))
	}()
	for i := range timeNear {
		current := t.n[i].clear()
		if current != nil {
			t.dispatchList(current, nil)
		}
	}
	for i := range 4 {
		for j := range timeLevel {
			current := t.t[i][j].clear()
			if current != nil {
				t.dispatchList(current, nil)
			}
		}
	}
}
