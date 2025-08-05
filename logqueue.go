package ltask

import (
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/phuslu/log"
)

type logMessage struct {
	timestamp int64
	id        serviceId
	sz        int64
	msg       unsafe.Pointer
}

type logItem struct {
	next *logItem
	msg  logMessage
}

type logQueue struct {
	freeList *logItem
	head     *logItem
	tail     *logItem
	l        int32
}

func newLogQueue() *logQueue {
	var q *logQueue
	q = (*logQueue)(malloc.Alloc(uint(unsafe.Sizeof(*q))))
	atomic.StoreInt32(&q.l, 0)

	return q
}

func (q logQueue) acquireLock() {
	for !atomic.CompareAndSwapInt32(&q.l, 0, 1) {
		log.Debug().Msgf("logQueue acquireLock failed, waiting...%d", q.l)
		runtime.Gosched()
	}
}

func (q logQueue) releaseLock() {
	atomic.StoreInt32(&q.l, 0)
}

func (q *logQueue) allocItem() (ret *logItem) {
	ret = q.freeList
	if ret == nil {
		ret = (*logItem)(malloc.Alloc(uint(unsafe.Sizeof(*ret))))
	} else {
		q.freeList = ret.next
	}

	return
}

func (q *logQueue) push(m *logMessage) (ok bool) {
	q.acquireLock()
	defer q.releaseLock()

	item := q.allocItem()
	if item == nil {
		return
	}

	ok = true

	if q.head == nil {
		q.head, q.tail = item, item
		item.next = nil
	} else {
		q.tail.next = item
		item.next = nil
		q.tail = item
	}
	item.msg = *m

	return
}

func (q *logQueue) pop() (m *logMessage, ok bool) {
	q.acquireLock()
	defer q.releaseLock()

	if q.head == nil {
		return
	}

	ok = true

	item := q.head
	q.head = item.next
	if q.head == nil {
		q.tail = nil
	}

	item.next = q.freeList
	q.freeList = item
	m = &logMessage{}
	*m = item.msg

	return
}

func (q *logQueue) freeItems(item *logItem) {
	for item != nil {
		var temp *logItem
		temp, item = item, item.next
		malloc.Free(unsafe.Pointer(temp))
	}
}

func (q *logQueue) delete() {
	for {
		m, ok := q.pop()
		if !ok {
			break
		}
		malloc.Free(unsafe.Pointer(m.msg))
	}
	q.freeItems(q.freeList)
	malloc.Free(unsafe.Pointer(q))
}
