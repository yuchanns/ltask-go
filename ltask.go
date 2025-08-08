package ltask

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/phuslu/log"
	"go.yuchanns.xyz/lua"
	"go.yuchanns.xyz/timefall"
	"go.yuchanns.xyz/xxchan"
)

func init() {
	log.DefaultLogger = log.Logger{
		Level:      log.DebugLevel,
		Caller:     1,
		TimeFormat: "2006-01-02 15:04:05.00",
		Writer: &log.ConsoleWriter{
			ColorOutput: false,
			Formatter: func(w io.Writer, a *log.FormatterArgs) (n int, err error) {
				return fmt.Fprintf(w, "[%s][%s]( %s ) %s\n%s", a.Time, strings.ToUpper(a.Level), a.Caller, a.Message, a.Stack)
			},
			Writer: os.Stderr,
		},
	}
}

var luaLib *lua.Lib

func OpenLibs(L *lua.State, lib *lua.Lib) {
	_ = L.GetGlobal("package")
	_, _ = L.GetField(-1, "preload")

	l := []*lua.Reg{
		{Name: "ltask.bootstrap", Func: ltaskBootstrapOpen},
	}
	L.SetFuncs(l, 0)
	L.Pop(2)

	luaLib = lib
}

func ltaskOpen(L *lua.State) int {
	l := []*lua.Reg{
		{Name: "pack", Func: LuaSerdePack},
		{Name: "unpack", Func: LuaSerdeUnpack},
		{Name: "remove", Func: LuaSerdeRemove},
		{Name: "unpack_remove", Func: LuaSerdeUnpackRemove},
		{Name: "timer_sleep", Func: ltaskSleep},
	}

	L.NewLib(l)

	l2 := []*lua.Reg{
		{Name: "send_message", Func: lsendMessage},
		{Name: "recv_message", Func: lrecvMessage},
		{Name: "message_receipt", Func: lmessageReceipt},
		{Name: "self", Func: lself},
		{Name: "timer_add", Func: ltaskTimerAdd},
		{Name: "timer_update", Func: ltaskTimerUpdate},
		{Name: "now", Func: ltaskNow},
		{Name: "label", Func: ltaskLabel},
		{Name: "pushlog", Func: ltaskPushLog},
		{Name: "poplog", Func: ltaskPopLog},
		{Name: "eventinit", Func: ltaskEventInit},
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

func ltaskSleep(L *lua.State) int {
	csec := L.OptInteger(1, 0)
	time.Sleep(centisecond * time.Duration(csec))
	return 0
}

func ltaskEventInit(L *lua.State) int {
	s := getS(L)
	index := s.task.services.getSockevent(s.id)
	if index >= 0 {
		return L.Errorf("Already init event")
	}
	index = int64(s.task.allocSockevent())
	if index < 0 {
		return L.Errorf("Too many sockevents")
	}
	// TODO: open sockevents
	s.task.services.initSockevent(s.id, index)
	return 0
}

func ltaskTimerAdd(L *lua.State) int {
	s := getS(L)
	t := s.task.timer
	if t == nil {
		return L.Errorf("Init timer before bootstrap")
	}
	ev := &timerEvent{
		session: int32(L.CheckInteger(1)),
		id:      s.id,
	}
	ti := L.CheckInteger(2)
	if ti < 0 || ti != int64(int32(ti)) {
		return L.Errorf("Invalid timer time: %d", ti)
	}
	t.Add(ev, time.Duration(ti)*centisecond)
	return 0
}

func ltaskTimerUpdate(L *lua.State) int {
	s := getS(L)
	t := s.task.timer
	if t == nil {
		return L.Errorf("Init timer before bootstrap")
	}
	if L.GetTop() > 1 {
		L.SetTop(1)
		L.CheckType(1, lua.LUA_TTABLE)
	}
	var idx int64
	t.Update(func(event *timerEvent) {
		// Pack session and id into a single int64 value
		v := int64(event.session)<<32 | int64(event.id)
		L.PushInteger(v)
		idx++
		L.SetI(1, int64(idx))
	})
	// Fill the rest of the table with nils
	// if the number of timer events is less than cap of the table,
	// so there are no old values left.
	// { 1, 2, 3, 4, 5 } => { 6, 7, 8, nil, nil }
	n := int64(L.RawLen(1))
	for i := int64(idx + 1); i <= n; i++ {
		L.PushNil()
		L.SetI(1, i)
	}
	return 1
}

func ltaskNow(L *lua.State) int {
	s := getS(L)
	t := s.task.timer
	if t == nil {
		return L.Errorf("Init timer before bootstrap")
	}
	start := t.Start()
	now := t.Now()
	L.PushInteger(start + now/100)
	L.PushInteger(start*100 + now)
	return 2
}

func lself(L *lua.State) int {
	s := getS(L)
	L.PushInteger(int64(s.id))
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
	data := L.ToUserData(1)
	sz := L.CheckInteger(2)
	s := getS(L)
	if !s.task.pushLog(s.id, data, sz) {
		return L.Errorf("log error")
	}

	return 0
}

func ltaskPopLog(L *lua.State) int {
	s := getS(L)
	m, ok := s.task.lqueue.pop()
	if !ok {
		return 0
	}
	// TODO: timer_starttime
	L.PushInteger(m.timestamp)
	L.PushInteger(int64(m.id))
	L.PushLightUserData(m.msg)
	L.PushInteger(m.sz)
	return 4
}

type ltask struct {
	config              *ltaskConfig
	workers             []workerThread
	eventInit           [maxSockEvent]atomicInt
	event               [maxSockEvent]*xxchan.Channel[struct{}]
	services            *servicePool
	schedule            *xxchan.Channel[int]
	timer               *timefall.Timer[timerEvent]
	lqueue              *logQueue
	externalMessage     *xxchan.Channel[unsafe.Pointer]
	externalLastMessage *message
	scheduleOwner       atomicInt
	activeWorker        atomicInt
	threadCount         atomicInt
	blockedService      int64
	// TODO: logfile?
}

func (task *ltask) allocSockevent() (index int) {
	for i := 0; i < maxSockEvent; i++ {
		if atomic.CompareAndSwapInt32(&task.eventInit[i], 0, 1) {
			return i
		}
	}
	return -1
}

func (task *ltask) pushLog(id serviceId, data unsafe.Pointer, sz int64) (ok bool) {
	now := time.Now()
	sec := now.Unix()
	nsec := now.Nanosecond()
	csec := sec*100 + int64(nsec/10_000_000)
	return task.lqueue.push(&logMessage{
		id:  id,
		msg: data,
		sz:  sz,
		// TODO: use timer_now
		timestamp: csec,
	})
}

func (task *ltask) init(L *lua.State, config *ltaskConfig) {
	task = (*ltask)(L.NewUserDataUv(int(unsafe.Sizeof(*task)), 0))
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_GLOBAL")
	task.lqueue = newLogQueue()
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

	atomic.StoreInt32(&task.scheduleOwner, threadNone)
	atomic.StoreInt32(&task.activeWorker, 0)
	atomic.StoreInt32(&task.threadCount, 0)

	for i := range task.event {
		ptr := malloc.Alloc(uint(xxchan.Sizeof[struct{}](1)))
		ch := xxchan.Make[struct{}](ptr, 1)
		task.event[i] = ch
		atomic.StoreInt32(&task.eventInit[i], 0)
	}
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
		worker := &task.workers[id]
		worker.init(task, serviceId(id))
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
