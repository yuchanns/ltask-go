package ltask

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

type cursor uint64

func (c *cursor) load() (chunk, offset uint64) {
	v := atomic.LoadUint64((*uint64)(c))
	offset = v & 0xFFFFFFFF
	chunk = v >> 32
	return
}

func (c *cursor) add(size uint64) (chunk, offset uint64) {
	v := atomic.AddUint64((*uint64)(c), size)
	offset = v & 0xFFFFFFFF
	chunk = v >> 32
	return
}

func (c *cursor) incChunk(chunk, offset uint64) bool {
	return atomic.CompareAndSwapUint64((*uint64)(c), chunk<<32|offset, (chunk+1)<<32)
}

func (c *cursor) reset(chunk, offset uint64) bool {
	return atomic.CompareAndSwapUint64((*uint64)(c), chunk<<32|offset, 0)
}

type arena struct {
	lock      sync.Mutex
	chunkSize uint64
	limit     int

	cursor cursor
	chunks [][]byte
}

func createArena(chunkSize uint64) *arena {
	if chunkSize > 0x7FFFFFFF {
		panic("chunk size too large")
	}
	a := &arena{
		chunkSize: chunkSize,
		chunks:    [][]byte{make([]byte, chunkSize)},
	}
	return a
}

func makeSlice[T any](a *arena, n uint64) []T {
	var t T
	return unsafe.Slice((*T)(malloc.alloc(uint64(unsafe.Sizeof(t)))), n)
}

func alloc[T any](a *arena) *T {
	var t T
	return (*T)(a.alloc(uint64(unsafe.Sizeof(t))))
}

func (a *arena) alloc(n uint64) unsafe.Pointer {
	chunk, next := a.cursor.add(n)
	if next < a.chunkSize {
		return unsafe.Pointer(&a.chunks[chunk][next-n : next][0])
	}
	return a.resize(chunk, next, n)
}

func (a *arena) resize(chunk, cursor, n uint64) unsafe.Pointer {
	a.lock.Lock()                              // Note that we don't defer Unlock here because resize is called recursively
	if a.limit != 0 && int(chunk) >= a.limit { //nolint:gosec
		a.lock.Unlock()
		panic(fmt.Sprintf("arena limit of %d chunks reached", a.limit))
	}
	// Check that another thread hasn't already resized the arena.
	if actualChunk, actualCursor := a.cursor.load(); actualChunk != chunk || actualCursor != cursor {
		a.lock.Unlock()
		return a.alloc(n)
	}

	// At this point we can't recurse, so we can defer the unlock.

	if chunk >= uint64(len(a.chunks)-1) && (a.limit == 0 || chunk+1 < uint64(a.limit)) { //nolint:gosec
		a.chunks = append(a.chunks, make([]byte, a.chunkSize))
	}
	if !a.cursor.incChunk(chunk, cursor) {
		a.lock.Unlock()
		return a.alloc(n)
	}
	defer a.lock.Unlock()
	cursor = n
	if cursor > a.chunkSize {
		panic(fmt.Sprintf("object size %d is larger than chunk size %d", n, a.chunkSize))
	}
	return unsafe.Pointer(&a.chunks[chunk][cursor-n : cursor][0])
}

func (a *arena) reset() {
	a.lock.Lock()
	defer a.lock.Unlock()
	beforeChunk, beforeCursor := a.cursor.load()
	// Zero the chunks.
	for _, chunk := range a.chunks {
		for i := range chunk {
			chunk[i] = 0
		}
	}
	if !a.cursor.reset(beforeChunk, beforeCursor) {
		panic("reset failed, another thread is using the arena")
	}
}
