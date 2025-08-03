package ltask

import (
	"sync/atomic"
	"unsafe"

	"github.com/phuslu/log"
	"go.yuchanns.xyz/lua"
	"go.yuchanns.xyz/xxchan"
)

var luaLib *lua.Lib

func OpenLibs(L *lua.State, lib *lua.Lib) {
	_ = L.GetGlobal("package")
	_, _ = L.GetField(-1, "preload")

	l := []luaLReg{
		{"ltask.bootstrap", ltaskBootstrapOpen},
	}
	luaLSetFuncs(L, l)
	L.Pop(2)

	luaLib = lib
}

func ltaskOpen(L *lua.State) int {
	l := []*lua.Reg{
		{"pack", LuaSerdePack},
		{"unpack", LuaSerdeUnpack},
		{"remove", LuaSerdeRemove},
		{"unpack_remove", LuaSerdeUnpackRemove},
		// timer_sleep
	}

	L.NewLib(l)
	return 1
}

type luaLReg struct {
	name string
	fn   lua.GoFunc
}

func luaLNewLib(L *lua.State, l []luaLReg) {
	L.NewTable()

	luaLSetFuncs(L, l)
}

func luaLSetFuncs(L *lua.State, l []luaLReg) {
	for _, i := range l {
		L.PushGoFunction(i.fn)
		L.SetField(-2, i.name)
	}
}

type ltask struct {
	config    *ltaskConfig
	workers   []workerThread
	eventInit []atomicInt
	event     []*xxchan.Channel[struct{}]
	services  *servicePool
	schedule  *xxchan.Channel[int]
	timer     *timer
	// TODO: logqueue?
	externalMessage     *xxchan.Channel[unsafe.Pointer]
	externalLastMessage *message
	scheduleOwner       atomicInt
	activeWorker        atomicInt
	threadCount         atomicInt
	blockedService      int64
	// TODO: logfile?
}

func (task *ltask) init(L *lua.State, config *ltaskConfig) {
	task = (*ltask)(L.NewUserDataUv(int(unsafe.Sizeof(*task)), 0))
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_GLOBAL")
	// task.logqueue
	task.config = config

	task.initWorker(L)

	task.services = newServicePool(config)
	ptr := malloc.Alloc(uint(xxchan.Sizeof[int](int(config.maxService))))
	task.schedule = xxchan.Make[int](ptr, int(config.maxService))
	// Windows compatiblity: initialize the timer with a nil value
	// to clear any wired data in the memory.
	task.timer = nil
	task.externalMessage = nil

	if config.externalQueue > 0 {
		ptr := malloc.Alloc(uint(xxchan.Sizeof[unsafe.Pointer](int(config.externalQueue))))
		task.externalMessage = xxchan.Make[unsafe.Pointer](ptr, int(config.externalQueue))
	}

	typ, _ := L.GetField(1, "debuglog")
	if typ == lua.LUA_TSTRING {
		logFile := L.ToString(-1)
		if logFile != "=" {
			// TODO: use file logger
		}
	} else {
		log.DefaultLogger.SetLevel(log.InfoLevel)
	}

	atomic.StoreInt64(&task.scheduleOwner, threadNone)
	atomic.StoreInt64(&task.activeWorker, 0)
	atomic.StoreInt64(&task.threadCount, 0)

	event := make([]*xxchan.Channel[struct{}], maxSockEvent)
	eventInit := make([]atomicInt, maxSockEvent)
	task.eventInit = eventInit
	for i := range event {
		ptr := malloc.Alloc(uint(xxchan.Sizeof[struct{}](1)))
		ch := xxchan.Make[struct{}](ptr, 1)
		event[i] = ch
		atomic.StoreInt64(&task.eventInit[i], 0)
	}
	task.event = event
}

func (task *ltask) initWorker(L *lua.State) {
	workerSize := int(unsafe.Sizeof(workerThread{}))

	workers := unsafe.Slice(
		(*workerThread)(L.NewUserDataUv(
			workerSize*int(task.config.worker), 0,
		)),
		task.config.worker,
	)
	task.workers = workers
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_WORKERS")

	for id := range task.config.worker {
		task.workers[id].init(task, id)
	}
}

type serviceUd struct {
	task *ltask
	id   serviceId
}

const (
	threadNone = -1
)

func getErrorMessage(L *lua.State) string {
	switch L.Type(-1) {
	case lua.LUA_TLIGHTUSERDATA:
		return *(*string)(L.ToUserData(-1))
	case lua.LUA_TSTRING:
		return L.ToString(-1)
	}
	return "Invalid error message"
}
