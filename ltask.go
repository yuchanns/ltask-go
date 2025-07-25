package ltask

import (
	"sync/atomic"
	"unsafe"

	"go.yuchanns.xyz/lua"
)

func OpenLibs(L *lua.State) {
	_ = L.GetGlobal("package")
	_, _ = L.GetField(-1, "preload")

	l := []luaLReg{
		{"ltask.bootstrap", ltaskBootstrap},
	}
	luaLSetFuncs(L, l)
	L.Pop(2)
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
	eventInit [maxSockEvent]atomicInt
	// TODO: event sockevent?
	services *servicePool
	schedule chan any
	// TODO: timer timerwheel?
	// TODO: logqueue?
	externalMessage     chan message
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
	if config.externalQueue > 0 {
		// FIXME: allocate a queue without gc tracking
		task.externalMessage = make(chan message, config.externalQueue)
	}

	atomic.StoreInt32(&task.scheduleOwner, threadNone)
	atomic.StoreInt32(&task.activeWorker, 0)
	atomic.StoreInt32(&task.threadCount, 0)
}

func (task *ltask) initWorker(L *lua.State) {
	task.workers = unsafe.Slice(
		(*workerThread)(L.NewUserDataUv(
			int(unsafe.Sizeof(workerThread{}))*int(task.config.worker), 0,
		)),
		task.config.worker,
	)
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
		// FIXME: allocate a queue without gc tracking
		worker.trigger = make(chan struct{}, 1)
	}
}

type serviceUd struct {
	task *ltask
	id   serviceId
}

const (
	threadNone = -1
)
