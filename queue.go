package ltask

import (
	"sync/atomic"
	"unsafe"
)

func isPow2(x int) bool {
	return x&(x-1) == 0
}

type queue struct {
	size int
	head atomicInt
	tail atomicInt
	data [][1]any
}

func newQueue(size int, stride int) (q *queue) {
	if !isPow2(size) {
		panic("Queue size must be a power of 2")
	}
	q = (*queue)(malloc.alloc(
		uint64(unsafe.Sizeof(queue{})) + uint64(size*stride),
	))
	q.size = size
	atomic.StoreInt32(&q.head, 0)
	atomic.StoreInt32(&q.tail, 0)
	return
}

func newQueueInt(size int) (q *queue) {
	return newQueue(size, int(unsafe.Sizeof(int(0))))
}

func newQueuePtr(size int) (q *queue) {
	return newQueue(size, int(unsafe.Sizeof(unsafe.Pointer(nil))))
}
