package ltask

import (
	"math/bits"
	"runtime"

	"go.yuchanns.xyz/lua"
)

const (
	defaultMaxService   = 65536
	defaultQueue        = 4096
	defaultQueueSending = defaultQueue
	maxWorker           = 256
)

func alignPow2(x uint) uint {
	if x <= 1 {
		return 1
	}
	return 1 << bits.Len(x-1)
}

type ltaskConfig struct {
	Worker        int
	Queue         int
	QueueSending  int
	MaxService    int
	ExternalQueue int
	CrashLog      [128]*string
}

func (config *ltaskConfig) Load(L *lua.State, index int) {
	L.CheckType(index, lua.LUA_TTABLE)
	config.Worker = configGetInit(L, index, "worker", 1)
	ncores := runtime.NumCPU()
	if ncores <= 1 {
		L.Errorf("Need at least 2 cores")
		return
	}
	if config.Worker == 0 {
		config.Worker = ncores - 1
	}
	if config.Worker > maxWorker {
		config.Worker = maxWorker
	}
	config.Queue = int(alignPow2(uint(configGetInit(L, index, "queue", defaultQueue))))
	config.QueueSending = int(alignPow2(uint(configGetInit(L, index, "queue_sending", defaultQueueSending))))
	config.MaxService = int(alignPow2(uint(configGetInit(L, index, "max_service", defaultMaxService))))
	config.ExternalQueue = configGetInit(L, index, "external_queue", 0)
	typ, _ := L.GetField(index, "crash_log")
	if typ != lua.LUA_TSTRING {
		config.CrashLog[0] = nil
	} else {
		var sz int
		log := L.ToLString(-1, &sz)
		if sz > len(config.CrashLog) {
			config.CrashLog[0] = nil
		} else {
			config.CrashLog[0] = new(string)
			*config.CrashLog[0] = log
		}
	}

	L.PushInteger(int64(config.Worker))
	L.SetField(index, "worker")
	L.PushInteger(int64(config.Queue))
	L.SetField(index, "queue")
	L.PushInteger(int64(config.MaxService))
	L.SetField(index, "max_service")
	L.PushValue(index)
}

func configGetInit(L *lua.State, index int, key string, opt int) int {
	typ, _ := L.GetField(index, key)
	if typ == lua.LUA_TNIL {
		L.Pop(1)
		return opt
	}
	if !L.IsInteger(-1) {
		return L.Errorf("%s should be an integer", key)
	}
	r := L.ToInteger(-1)
	L.Pop(1)
	return int(r)
}
