package ltask

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"go.yuchanns.xyz/lua"
)

var luaLib *lua.Lib

func OpenLibs(L *lua.State, lib *lua.Lib) {
	_ = L.GetGlobal("package")
	_, _ = L.GetField(-1, "preload")

	l := []luaLReg{
		{"ltask.bootstrap", ltaskBootstrap},
	}
	luaLSetFuncs(L, l)
	L.Pop(2)

	luaLib = lib
}

func openLtask(L *lua.State) int {
	return 0
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
	event     []chan struct{}
	services  *servicePool
	schedule  chan int
	timer     *timer
	// TODO: logqueue?
	externalMessage     chan any
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
	task.schedule = make(chan int, config.maxService)
	if config.externalQueue > 0 {
		task.externalMessage = make(chan any, config.externalQueue)
	}

	atomic.StoreInt32(&task.scheduleOwner, threadNone)
	atomic.StoreInt32(&task.activeWorker, 0)
	atomic.StoreInt32(&task.threadCount, 0)

	event := make([]chan struct{}, maxSockEvent)
	eventInit := make([]atomicInt, maxSockEvent)
	task.eventInit = eventInit
	for i := range event {
		ch := make(chan struct{})
		event[i] = ch
		atomic.StoreInt32(&task.eventInit[i], 0)
	}
	task.event = event
	// Windows compatiblity: initialize the timer with a nil value
	// to clear any wired data in the memory.
	task.timer = nil
}

func (task *ltask) initWorker(L *lua.State) {
	workers := unsafe.Slice(
		(*workerThread)(L.NewUserDataUv(
			int(unsafe.Sizeof(workerThread{}))*int(task.config.worker), 0,
		)),
		task.config.worker,
	)
	task.workers = workers
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_WORKERS")

	for id := range task.config.worker {
		worker := &task.workers[id]
		worker.task = task
		worker.workerId = id
		worker.running = 0
		worker.binding = 0
		worker.waiting = 0
		atomic.StoreInt32(&worker.serviceReady, 0)
		atomic.StoreInt32(&worker.serviceDone, 0)
		worker.termSignal = 0
		worker.sleeping = 0
		worker.wakeup = 0
		worker.busy = 0

		worker.trigger = &sync.Cond{L: &sync.Mutex{}}
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
