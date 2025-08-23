package ltask

import (
	"embed"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"go.yuchanns.xyz/lua"
	"go.yuchanns.xyz/timefall"
)

func getPtr[T any](L *lua.State, key string) *T {
	typ := L.GetField(lua.LUA_REGISTRYINDEX, key)
	if typ == lua.LUA_TNIL {
		L.Errorf("%s is absense", key)
		return nil
	}
	v := L.ToUserData(-1)
	if v == nil {
		L.Errorf("Invalid %s", key)
		return nil
	}
	L.Pop(1)

	return (*T)(v)
}

//go:embed lualib/*.lua service/*.lua
var embedfs embed.FS

var embedfsList = []*embed.FS{
	&embedfs,
}

// UseEmbedFS allows you to add additional embed.FS instances to the ltask runtime.
// Calling it before `ltask.OpenLib`. It is not thread-safe.
// This is useful if you want to embed your own Lua scripts as built-in services.
// To access these files, use:
// - `require("ltask").searchpath("yourfile.lua")`
// - `require("ltask").loadfile("yourfile.lua")`
// - `require("ltask.bootstrap").searchpath("yourfile.lua")`
// - `require("ltask.bootstrap").loadfile("yourfile.lua")`
// - `require("ltask.bootstrap").dofile("yourfile.lua")`
// - `require("ltask.bootstrap").readfile("yourfile.lua")`
func UseEmbedFS(fs ...*embed.FS) {
	embedfsList = append(embedfsList, fs...)
}

func ltaskInit(L *lua.State) int {
	if L.GetTop() == 0 {
		L.NewTable()
	}
	typ := L.GetField(lua.LUA_REGISTRYINDEX, "LTASK_CONFIG")
	if typ != lua.LUA_TNIL {
		return L.Errorf("Already init")
	}
	L.Pop(1)

	var config *ltaskConfig
	config = (*ltaskConfig)(L.NewUserDataUv(int(unsafe.Sizeof(*config)), 0))
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_CONFIG")

	config.load(L, 1)

	if config.crashLog != nil {
		// TODO: set crash log
	}

	var task *ltask
	luaLib := (*lua.Lib)(L.ToUserData(L.UpValueIndex(1)))
	task.init(L, config, luaLib)

	return 1
}

func ltaskBootPushLog(L *lua.State) int {
	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	data := L.ToUserData(1)
	sz := L.CheckInteger(2)
	if !task.pushLog(serviceIdSystem, data, sz) {
		return L.Errorf("log error")
	}

	return 0
}

type timerEvent struct {
	session session
	id      serviceId
}

type timerUpdateUd struct {
	L *lua.State
	n int
}

var centisecond = time.Duration(10 * time.Millisecond)

func ltaskInitTimer(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	if task.timer != nil {
		return L.Errorf("Timer can init only once")
	}
	task.timer = timefall.New[timerEvent](centisecond)

	return 0
}

func ltaskNewService(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	label := L.CheckString(1)
	source := L.CheckString(2)
	chunkName := L.CheckString(3)
	sid := serviceId(L.OptInteger(4, 0))
	workerId := int32(L.OptInteger(5, -1))

	id := task.services.newService(sid)

	if !task.initService(L, id, label, source, chunkName, workerId) {
		L.PushBoolean(false)
		L.Insert(-2)
		return 2
	}

	L.PushInteger(int64(id))
	return 1
}

func ltaskInitRoot(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	var id = serviceId(L.CheckInteger(1))
	if id != serviceIdRoot {
		return L.Errorf("Id should be ROOT(1)")
	}
	s := task.services.getService(id)
	if s == nil {
		return L.Errorf("Service %d not found", id)
	}
	if !s.requiref("ltask.root", ltaskRootOpen, L) {
		return L.Errorf("Require ltask.root failed: %s", getErrorMessage(L))
	}
	return 0
}

func checkField(L *lua.State, index int, key string) int64 {
	typ := L.GetField(index, key)
	if typ != lua.LUA_TNUMBER {
		return int64(L.Errorf(".%s should be an integer", key))
	}
	v := L.ToInteger(-1)
	L.Pop(1)
	return v
}

func lmessageReceipt(L *lua.State) int {
	s := getS(L)
	receipt, m := s.task.services.readReceipt(s.id)
	if receipt == messageReceiptNone {
		return L.Errorf("No receipt")
	}
	L.PushInteger(receipt)
	if m == nil {
		return 1
	}
	if receipt == messageReceiptResponse {
		// only for schedule message NEW
		L.PushInteger(int64(m.to))
		m.delete()
		return 2
	}
	if m.msg == nil {
		m.delete()
		return 1
	}
	L.PushLightUserData(m.msg)
	L.PushInteger(m.sz)
	m.delete()

	return 3
}

func lsendMessage(L *lua.State) int {
	s := getS(L)
	msg := genSendMessage(L, s.id)
	if !L.IsYieldable() {
		msg.delete()
		return L.Errorf("Cannot send message in none-yieldable context")
	}
	if !s.task.services.sendMessage(s.id, msg) {
		msg.delete()
		return L.Errorf("Cannot send message")
	}

	return 0
}

func lrecvMessage(L *lua.State) (r int) {
	s := getS(L)
	m := s.task.services.popMessage(s.id)
	if m == nil {
		return
	}
	r = 3
	L.PushInteger(int64(m.from))
	L.PushInteger((int64(m.session)))
	L.PushInteger(int64(m.typ))
	if m.msg != nil {
		L.PushLightUserData(m.msg)
		L.PushInteger(m.sz)
		r += 2
	}
	m.delete()
	return
}

func lpostMessage(L *lua.State) int {
	L.CheckType(1, lua.LUA_TTABLE)
	msg := newMessage(&message{
		from:    int32(checkField(L, 1, "from")),
		to:      int32(checkField(L, 1, "to")),
		session: session(checkField(L, 1, "session")),
		typ:     int(checkField(L, 1, "type")),
	})
	typ := L.GetField(1, "message")
	if typ != lua.LUA_TNIL {
		if typ != lua.LUA_TLIGHTUSERDATA {
			return L.Errorf(".message should be a pointer")
		}
		msg.msg = L.ToUserData(-1)
		L.Pop(1)
		msg.sz = checkField(L, 1, "size")
	}
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	if !task.services.postMessage(msg) {
		msg.delete()
		return L.Errorf("push message failed")
	}
	task.checkMessageTo(msg.to)
	return 0
}

type taskContext struct {
	task *ltask
	wg   *sync.WaitGroup
}

func ltaskRun(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	var (
		useMainThread bool
		mainThreadId  int64
	)
	if L.IsInteger(1) {
		useMainThread = true
		mainThreadId = L.CheckInteger(1)
		if mainThreadId >= 0 && mainThreadId >= task.config.worker {
			return L.Errorf("Invalid mainthread %d", mainThreadId)
		}
	}

	var ctx *taskContext
	ctx = (*taskContext)(L.NewUserDataUv(int(unsafe.Sizeof(*ctx)), 0))

	ctx.task = task
	wg := &sync.WaitGroup{}
	ctx.wg = wg

	var mainThread *workerThread

	for i := range task.workers {
		if useMainThread && int64(i) == mainThreadId {
			mainThread = &task.workers[i]
			continue
		}

		wg.Add(1)
		go func(w *workerThread) {
			defer wg.Done()

			w.start()
		}(&task.workers[i])
	}

	if useMainThread && mainThread != nil {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		mainThread.start()
	}

	return 1
}

func ltaskWait(L *lua.State) int {
	L.CheckType(1, lua.LUA_TUSERDATA)
	ctx := (*taskContext)(L.ToUserData(1))
	ctx.wg.Wait()

	ctx.task.lqueue.delete()
	for i := range ctx.task.event {
		ctx.task.event[i].close()
	}

	ctx.task.externalLastMessage = nil
	if ctx.task.externalMessage != nil {
		for ctx.task.externalMessage.Len() > 0 {
			ctx.task.externalMessage.Pop()
		}
		malloc.Free(unsafe.Pointer(ctx.task.externalMessage))
		ctx.task.externalMessage = nil
	}

	return 0
}

func ltaskDeinit(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")

	for i := range task.workers {
		w := &task.workers[i]
		w.destroy()
	}
	task.services.destroy()
	for task.schedule.Len() > 0 {
		task.schedule.Pop()
	}
	malloc.Free(unsafe.Pointer(task.schedule))
	task.schedule = nil
	task.timer.Destroy()

	L.PushNil()
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_GLOBAL")
	return 0
}

var bootInit atomic.Int32

func ltaskBootstrapOpen(L *lua.State) int {
	if bootInit.Add(1) != 1 {
		return L.Errorf("ltask.bootstrap can only require once")
	}
	l := []*lua.Reg{
		{Name: "searchpath", Func: ltaskSearchPath},
		{Name: "readfile", Func: ltaskReadFile},
		{Name: "loadfile", Func: ltaskLoadFile},
		{Name: "dofile", Func: ltaskDoFile},
		{Name: "deinit", Func: ltaskDeinit},
		{Name: "run", Func: ltaskRun},
		{Name: "wait", Func: ltaskWait},
		{Name: "post_message", Func: lpostMessage},
		{Name: "new_service", Func: ltaskNewService},
		{Name: "init_timer", Func: ltaskInitTimer},
		{Name: "init_root", Func: ltaskInitRoot},
		{Name: "pushlog", Func: ltaskBootPushLog},
		// We don't need `init_socket` here, as it is proceed by Go runtime automatically.
		{Name: "pack", Func: LuaSerdePack},
		{Name: "unpack", Func: LuaSerdeUnpack},
		{Name: "remove", Func: LuaSerdeRemove},
		{Name: "unpack_remove", Func: LuaSerdeUnpackRemove},
	}

	L.NewLib(l)

	L.PushLightUserData(L.ToUserData(L.UpValueIndex(1)))
	l2 := []*lua.Reg{
		{Name: "init", Func: ltaskInit},
	}
	L.SetFuncs(l2, 1)
	return 1
}
