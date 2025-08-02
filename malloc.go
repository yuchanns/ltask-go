package ltask

import (
	"sync/atomic"
	"unsafe"

	"github.com/smasher164/mem"
)

type Allocator interface {
	Alloc(size uint) unsafe.Pointer
	Free(ptr unsafe.Pointer)
}

var malloc Allocator

func init() {
	malloc = &defaultAllocator{}
}

type defaultAllocator struct{}

func (a *defaultAllocator) Alloc(size uint) unsafe.Pointer {
	return mem.Alloc(size)
}

func (a *defaultAllocator) Free(ptr unsafe.Pointer) {
	mem.Free(ptr)
}

var allocInit atomic.Int32

func SetAllocator(alloc Allocator) {
	if alloc == nil {
		panic("allocator cannot be nil")
	}
	if allocInit.Add(1) != 1 {
		panic("allocator can only be set once")
	}
	malloc = alloc
}
