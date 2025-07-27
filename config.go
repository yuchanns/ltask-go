package ltask

import (
	"math/bits"
	"runtime"

	"go.yuchanns.xyz/lua"
)

type atomicInt = int32

const (
	defaultMaxService   = 65536
	defaultQueue        = 4096
	defaultQueueSending = defaultQueue
	maxWorker           = 256
	maxSockEvent        = 16
)

func alignPow2(x uint) uint {
	if x <= 1 {
		return 1
	}
	return 1 << bits.Len(x-1)
}

type ltaskConfig struct {
	worker        int64
	queue         int64
	queueSending  int64
	maxService    int64
	externalQueue int64
	crashLog      []byte
}

func (config *ltaskConfig) load(L *lua.State, index int) {
	L.CheckType(index, lua.LUA_TTABLE)
	config.worker = configGetInit(L, index, "worker", 0)
	ncores := runtime.NumCPU()
	if ncores <= 1 {
		L.Errorf("Need at least 2 cores")
		return
	}
	if config.worker == 0 {
		config.worker = int64(ncores - 1)
	}
	if config.worker > maxWorker {
		config.worker = maxWorker
	}
	config.queue = int64(alignPow2(uint(configGetInit(L, index, "queue", defaultQueue))))
	config.queueSending = int64(alignPow2(uint(configGetInit(L, index, "queue_sending", defaultQueueSending))))
	config.maxService = int64(alignPow2(uint(configGetInit(L, index, "max_service", defaultMaxService))))
	config.externalQueue = configGetInit(L, index, "external_queue", 0)
	typ, _ := L.GetField(index, "crash_log")
	if typ == lua.LUA_TSTRING {
		var size int
		log := L.ToLString(-1, &size)
		if size < 128 {
			crashLog := make([]byte, size+1)
			copy(crashLog, []byte(log))
			config.crashLog = crashLog
		}
	}

	L.PushInteger(int64(config.worker))
	L.SetField(index, "worker")
	L.PushInteger(int64(config.queue))
	L.SetField(index, "queue")
	L.PushInteger(int64(config.maxService))
	L.SetField(index, "max_service")
	L.PushValue(index)
}

func configGetInit(L *lua.State, index int, key string, opt int64) int64 {
	typ, _ := L.GetField(index, key)
	if typ == lua.LUA_TNIL {
		L.Pop(1)
		return opt
	}
	if !L.IsInteger(-1) {
		return int64(L.Errorf("%s should be an integer", key))
	}
	r := L.ToInteger(-1)
	L.Pop(1)
	return r
}
