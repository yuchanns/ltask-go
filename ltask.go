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

	l := []*lua.Reg{
		{"ltask.bootstrap", ltaskBootstrapOpen},
	}
	L.SetFuncs(l, 0)
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

	l2 := []*lua.Reg{
		{"recv_message", lrecvMessage},
		{"self", lself},
		{"label", ltaskLabel},
		{"pushlog", ltaskPushLog},
	}

	typ, _ := L.GetField(lua.LUA_REGISTRYINDEX, "LTASK_ID")
	if typ != lua.LUA_TLIGHTUSERDATA {
		L.Errorf("No service id, the VM is not inited by ltask")
	}
	ud := L.ToUserData(-1)
	L.Pop(1)

	L.PushLightUserData(ud)
	L.SetFuncs(l2, 1)

	return 1
}

func getS(L *lua.State) *serviceUd {
	ud := L.ToUserData(L.UpValueIndex(1))
	if ud == nil {
		panic("Invalid service userdata")
	}
	return (*serviceUd)(ud)
}

func lself(L *lua.State) int {
	s := getS(L)
	L.PushInteger(s.id)
	return 1
}

func ltaskLabel(L *lua.State) int {
	s := getS(L)
	label := s.task.services.getLabel(s.id)
	L.PushString(label)
	return 1
}

func ltaskPushLog(L *lua.State) int {
	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	// data := L.ToUserData(1)
	// sz := L.CheckInteger(2)
	// s := getS(L)
	// TODO: logqueue
	//

	return 0
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
