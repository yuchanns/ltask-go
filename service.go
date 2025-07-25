package ltask

import (
	"go.yuchanns.xyz/lua"
)

const (
	typeIdCount = 6
)

type memoryStat struct {
	count [typeIdCount]int64
	mem   int64
	limit int64
}

type service struct {
	L             *lua.State
	rL            *lua.State
	msg           chan any
	out           *message
	bounce        *message
	status        int64
	receipt       int64
	bindingThread int64
	id            serviceId
	label         [32]byte
	stat          memoryStat
	cpucost       uint64
	clock         uint64
}

type servicePool struct {
	mask     int64
	queueLen int64
	id       uint64
	s        []*service
}

func newServicePool(config *ltaskConfig) (pool *servicePool) {
	pool = alloc[servicePool](malloc)
	pool.mask = config.maxService - 1
	pool.queueLen = config.queue
	pool.id = typeIdCount
	pool.s = makeSlice[*service](malloc, uint64(config.maxService))
	return
}
