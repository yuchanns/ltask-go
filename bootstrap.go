package ltask

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/phuslu/log"
	"go.yuchanns.xyz/lua"
)

func getPtr[T any](L *lua.State, key string) *T {
	typ, _ := L.GetField(lua.LUA_REGISTRYINDEX, key)
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

func ltaskInit(L *lua.State) int {
	if L.GetTop() == 0 {
		L.NewTable()
	}
	typ, _ := L.GetField(lua.LUA_REGISTRYINDEX, "LTASK_CONFIG")
	if typ != lua.LUA_TNIL {
		return L.Errorf("Already init")
	}
	L.Pop(1)

	var config *ltaskConfig
	config = (*ltaskConfig)(L.NewUserDataUv(int(unsafe.Sizeof(*config)), 0))
	_ = L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_CONFIG")

	config.load(L, 1)

	if config.crashLog != nil {
		// TODO: set crash log
	}

	var task *ltask
	task.init(L, config)

	return 1
}

func ltaskInitTimer(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	if task.timer != nil {
		return L.Errorf("Timer can init only once")
	}
	task.timer = newTimer()

	return 0
}

func ltaskNewService(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	label := L.CheckString(1)
	source := L.CheckString(2)
	chunkName := L.CheckString(3)
	sid := L.OptInteger(4, 0)
	workerId := L.OptInteger(5, -1)

	id := task.services.newService(sid)

	if !task.initService(L, id, label, source, chunkName, workerId) {
		L.PushBoolean(false)
		L.Insert(-2)
		return 2
	}

	L.PushInteger(id)
	return 1
}

func ltaskInitRoot(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	var id serviceId = L.CheckInteger(1)
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
	typ, _ := L.GetField(index, key)
	if typ != lua.LUA_TNUMBER {
		return int64(L.Errorf(".%s should be an integer", key))
	}
	v := L.ToInteger(-1)
	L.Pop(1)
	return v
}

func lpostMessage(L *lua.State) int {
	typ, _ := L.GetField(1, "type")
	L.CheckType(1, lua.LUA_TTABLE)
	msg := newMessage(&message{
		from:    checkField(L, 1, "from"),
		to:      checkField(L, 1, "to"),
		session: session(checkField(L, 1, "session")),
		typ:     typ,
	})
	typ, _ = L.GetField(1, "message")
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

			threadWorker(w)
		}(&task.workers[i])
	}

	if useMainThread && mainThread != nil {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		threadWorker(mainThread)
	}

	return 1
}

func ltaskWait(L *lua.State) int {
	L.CheckType(1, lua.LUA_TUSERDATA)
	ctx := (*taskContext)(L.ToUserData(1))
	ctx.wg.Wait()

	for i := range ctx.task.event {
		for ctx.task.event[i].Len() > 0 {
			ctx.task.event[i].Pop()
		}
		malloc.Free(unsafe.Pointer(ctx.task.event[i]))
		ctx.task.event[i] = nil
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
	task.timer.destroy()

	L.PushNil()
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_GLOBAL")
	return 0
}

func threadWorker(w *workerThread) {
	p := w.task.services
	atomic.AddInt64(&w.task.activeWorker, 1)
	log.Debug().Msgf("Worker %d start", w.workerId)

	for {
		if w.termSignal > 0 {
			break
		}
		id := w.getJob()
		var dead bool
		if id == 0 {
			// No job, try to acquire scheduler to find a job
			var noJob = true

			for {
				if !w.acquireScheduler() {
					continue
				}
				noJob = w.schedule()
				w.releaseScheduler()

				if w.serviceDone == 0 {
					break
				}
			}

			if noJob && w.task.blockedService == 0 {
				// go to sleep if no job and no blocked service
				atomic.AddInt64(&w.task.threadCount, -1)
				log.Debug().Msgf("Worker %d sleeping", w.workerId)
				w.sleep()
				atomic.AddInt64(&w.task.activeWorker, 1)
				log.Debug().Msgf("Worker %d wakeup", w.workerId)
			}
			continue
		}
		// Get a job to do
		w.busy = 1
		w.running = id
		if w.waiting == id {
			w.waiting = 0
		}
		status := p.getStatus(id)
		if status == serviceStatusDead {
			log.Debug().Msgf("Service %d is dead", id)
		} else {
			log.Debug().Msgf("Service %d is running on worker %d", id, w.workerId)
			if status != serviceStatusSchedule {
				panic("Service is not in schedule status")
			}
			p.setStatus(id, serviceStatusRunning)
			if !p.resume(id) {
				dead = true
				log.Debug().Msgf("Service %d quit", id)
				p.setStatus(id, serviceStatusDead)
				if id == serviceIdRoot {
					// root quit, wakeup others
					w.task.quitAllWorkers()
					w.task.wakeupAlWorkers()
					break
				}
				//
			} else {
				p.setStatus(id, serviceStatusDone)
			}
		}
		w.busy = 0

		// check binding
		if dead && w.binding == id {
			w.binding = 0
		} else if !dead && p.getBindingThread(id) == w.workerId {
			w.binding = id
		}

		for !w.completeJob() {
			// Unable to complete job (running -> done)
			// Try to acquire scheduler and then complete again
			if !w.acquireScheduler() {
				continue
			}
			if !w.completeJob() {
				// Still unable to complete job, try to dispatch
				w.dispatch()
				for !w.completeJob() {
				}
			}
			w.schedule()
			w.releaseScheduler()
			break
		}
	}
	w.quit()
	atomic.AddInt64(&w.task.threadCount, -1)
	log.Debug().Msgf("Worker %d quit", w.workerId)
}

var bootInit atomic.Int32

func ltaskBootstrapOpen(L *lua.State) int {
	if bootInit.Add(1) != 1 {
		return L.Errorf("ltask.bootstrap can only require once")
	}
	l := []luaLReg{
		{"init", ltaskInit},
		{"deinit", ltaskDeinit},
		{"run", ltaskRun},
		{"wait", ltaskWait},
		{"post_message", lpostMessage},
		{"new_service", ltaskNewService},
		{"init_timer", ltaskInitTimer},
		{"init_root", ltaskInitRoot},
		// We don't need `init_socket` here, as it is proceed by Go runtime automatically.
		{"pack", LuaSerdePack},
		{"unpack", LuaSerdeUnpack},
		{"remove", LuaSerdeRemove},
		{"unpack_remove", LuaSerdeUnpackRemove},
	}

	luaLNewLib(L, l)
	return 1
}
