package ltask

import (
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/phuslu/log"
	"github.com/smasher164/mem"
	"go.yuchanns.xyz/lua"
	"go.yuchanns.xyz/timefall"
	"go.yuchanns.xyz/xxchan"
)

type ltask struct {
	luaLib              *lua.Lib
	config              *ltaskConfig
	workers             []workerThread
	eventInit           [maxSockEvent]atomicInt
	event               [maxSockEvent]sockEvent
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
}

func (task *ltask) allocSockevent() (index int) {
	for i := range maxSockEvent {
		if atomic.CompareAndSwapInt32(&task.eventInit[i], 0, 1) {
			return i
		}
	}
	return -1
}

func (task *ltask) pushLog(id serviceId, data unsafe.Pointer, sz int64) (ok bool) {
	return task.lqueue.push(&logMessage{
		id:        id,
		msg:       data,
		sz:        sz,
		timestamp: task.timer.Now(),
	})
}

func (task *ltask) init(L *lua.State, config *ltaskConfig, luaLib *lua.Lib) {
	task = (*ltask)(L.NewUserDataUv(int(unsafe.Sizeof(*task)), 0))
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_GLOBAL")
	task.lqueue = newLogQueue()
	task.config = config

	task.initWorker(L)

	task.services = newServicePool(config)
	ptr := malloc.Alloc(uint(xxchan.Sizeof[int](int(config.maxService))))
	task.schedule = xxchan.Make[int](ptr, int(config.maxService))
	task.timer = nil
	task.externalMessage = nil
	task.luaLib = luaLib

	if config.externalQueue > 0 {
		ptr := malloc.Alloc(uint(xxchan.Sizeof[unsafe.Pointer](int(config.externalQueue))))
		task.externalMessage = xxchan.Make[unsafe.Pointer](ptr, int(config.externalQueue))
	}

	if L.GetField(1, "debuglog") == lua.LUA_TSTRING {
		logFile := L.ToString(-1)
		if logFile != "=" {
		}
	} else {
		log.DefaultLogger.SetLevel(log.InfoLevel)
	}

	atomic.StoreInt32(&task.scheduleOwner, threadNone)
	atomic.StoreInt32(&task.activeWorker, 0)
	atomic.StoreInt32(&task.threadCount, 0)

	for i := range task.event {
		task.event[i].init()
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

func (task *ltask) checkMessageTo(to serviceId) {
	p := task.services
	status := p.getStatus(to)
	if status == serviceStatusIdle {
		log.Debug().Msgf("Service %d is in schedule", to)
		p.setStatus(to, serviceStatusSchedule)
		task.scheduleBack(to)
		return
	}
	sockId := task.services.getSockevent(to)
	if sockId < 0 {
		return
	}
	log.Debug().Msgf("Trigger sockevent of service %d", to)
	task.event[sockId].trigger()
}

func (task *ltask) scheduleBack(id serviceId) {
	if !task.schedule.Push(int(id)) {
		panic("schedule channel is full")
	}
}

func (task *ltask) wakeupAllWorkers() {
	for i := range task.workers {
		task.workers[i].wake()
	}
}

func (task *ltask) quitAllWorkers() {
	for i := range task.workers {
		task.workers[i].termSignal = 1
	}
}

func (task *ltask) triggerAllSockevent() {
	for i := range task.event {
		task.event[i].trigger()
	}
}

func (task *ltask) dispatchExternalMessages() {
	var send bool
	if task.externalLastMessage != nil {
		if task.services.pushMessage(task.externalLastMessage.to, task.externalLastMessage) {
			return
		}
		task.externalLastMessage = nil
		send = true
	}
	for {
		msg, ok := task.externalMessage.Pop()
		if !ok {
			break
		}
		buf := serdePackString("external", msg)
		msg, sz := mallocFromBuffer(buf)
		m := newMessage(&message{
			from:    0,
			to:      1,
			session: 0,
			typ:     messageTypeRequest,
			msg:     msg,
			sz:      int64(sz),
		})
		if task.services.pushMessage(m.to, m) {
			task.externalLastMessage = m
			return
		}
		send = true
	}
	if send {
		task.checkMessageTo(1)
	}
}

func (task *ltask) collectDoneJobs() (done []serviceId) {
	for i := range task.workers {
		job := task.workers[i].doneJob()
		if job != 0 {
			log.Debug().Msgf("Worker %d done service %d", task.workers[i].workerId, job)
			done = append(done, job)
		}
	}
	return
}

func (task *ltask) triggerBlockedWorkers() {
	if task.blockedService == 0 {
		return
	}
	var blocked int64
	for i := range task.workers {
		w := &task.workers[i]
		if w.waiting == 0 {
			continue
		}
		running := w.running
		if running == 0 {
			blocked = 1
			continue
		}
		sockId := task.services.getSockevent(running)
		if sockId >= 0 {
			task.event[sockId].trigger()
		}
		w.waiting = 0
	}

	task.blockedService = blocked
}

func (task *ltask) assignPrepare(prepare []serviceId) {
	var (
		workerId   int
		useBusy    bool
		useBinding bool
	)

	for i := range prepare {
		id := prepare[i]
		for {
			if workerId >= len(task.workers) {
				if !useBusy {
					useBusy = true
					workerId = 0
				} else {
					useBinding = true
					workerId = 0
				}
			}
			w := &task.workers[workerId]
			workerId++
			if !(useBusy || w.busy == 0) || !(w.binding == 0 || useBinding) {
				continue
			}
			assign := w.assignJob(id)
			if assign == 0 {
				continue
			}
			w.wake()
			log.Debug().Msgf("Worker %d is assigned service %d", w.workerId, assign)
			if assign == id {
				break
			}
		}
	}
}

func (task *ltask) prepare(prepare []serviceId, freeSlots int) []serviceId {
	for range freeSlots {
		job, ok := task.schedule.Pop()
		if !ok {
			break
		}
		id := serviceId(job)
		worker := task.services.getBindingThread(id)
		if worker < 0 {
			prepare = append(prepare, id)
			continue
		}
		w := &task.workers[worker]
		if !w.bindingQueue.Push(id) {
			task.schedule.Push(job)
			continue
		}
		id = w.assignJob(id)
		if id == 0 {
			continue
		}
		w.kickRunning(id)
		w.wake()
		log.Debug().Msgf("Worker %d is assigned service %d", w.workerId, id)
		freeSlots--
	}
	return prepare
}

func (task *ltask) countFreeSlots() (slots int) {
	for i := range task.workers {
		w := &task.workers[i]
		if w.serviceReady != 0 {
			continue
		}
		q := w.bindingQueue
		id, ok := q.Pop()
		if !ok {
			slots++
			continue
		}
		atomic.StoreInt32(&w.serviceReady, id)
		w.kickRunning(id)
		w.wake()
		log.Debug().Msgf("Worker %d is assigned service %d from binding queue", w.workerId, id)
	}

	return
}

func (task *ltask) getPendingJobs() (pending []serviceId) {
	for i := range task.workers {
		w := &task.workers[i]
		if w.busy == 0 {
			continue
		}
		id := w.stolen()
		if id != 0 {
			pending = append(pending, id)
		}
	}

	return
}

func (task *ltask) dispatchOutMessages(doneJobs []serviceId) {
	p := task.services

	for i := range doneJobs {
		id := doneJobs[i]
		status := p.getStatus(id)
		msg := p.outMessage(id)
		if status == serviceStatusDead {
			if msg == nil || msg.to != serviceIdRoot || msg.typ != messageTypeSignal {
				panic("Service is dead but has no signal to root")
			}
			if s := p.getService(msg.to); s == nil || s.status == serviceStatusDead {
				log.Debug().Msgf("Root service is missing")
				p.deleteService(id)
				continue
			}
			if p.pushMessage(msg.to, msg) {
				log.Debug().Msgf("Root service is blocked, service %d will try to signal it later", id)
				task.scheduleBack(id)
				continue
			}
			log.Debug().Msgf("Service %d signaled root service", id)
			task.checkMessageTo(msg.to)
			continue
		}
		if msg != nil {
			task.dispatchOutMessage(id, msg)
		}
		if status != serviceStatusDone {
			panic("Service status is not done")
		}
		if !p.hasMessage(id) {
			sockId := p.getSockevent(id)
			if sockId >= 0 {
				log.Debug().Msgf("Service %d back to schedule", id)
				p.pushMessage(id, newMessage(&message{
					from:    serviceIdSystem,
					to:      id,
					session: 0,
					typ:     messageTypeIdle,
				}))
				p.setStatus(id, serviceStatusSchedule)
				task.scheduleBack(id)
				continue
			}
			log.Debug().Msgf("Service %d is idle", id)
			p.setStatus(id, serviceStatusIdle)
		} else {
			log.Debug().Msgf("Service %d back to schedule", id)
			p.setStatus(id, serviceStatusSchedule)
			task.scheduleBack(id)
		}
	}
}

func (task *ltask) dispatchScheduleMessage(id serviceId, msg *message) {
	p := task.services
	if id != serviceIdRoot {
		p.writeReceipt(id, messageReceiptError, msg)
		return
	}
	sid := msg.session
	switch msg.typ {
	case messageScheduleNew:
		msg.to = p.newService(sid)
		log.Debug().Msgf("New service %d", msg.to)
		if msg.to == 0 {
			p.writeReceipt(id, messageReceiptError, msg)
		} else {
			p.writeReceipt(id, messageReceiptResponse, msg)
		}
	case messageScheduleDel:
		log.Debug().Msgf("Delete service %d", sid)
		p.deleteService(serviceId(sid))
		msg.delete()
		p.writeReceipt(id, messageReceiptDone, nil)
	default:
		p.writeReceipt(id, messageReceiptError, msg)
	}
}

func (task *ltask) dispatchOutMessage(id serviceId, msg *message) {
	p := task.services
	if msg.to == serviceIdSystem {
		task.dispatchScheduleMessage(id, msg)
		return
	}
	if s := p.getService(msg.to); s == nil || s.status == serviceStatusDead {
		p.writeReceipt(id, messageReceiptError, msg)
	} else if p.pushMessage(msg.to, msg) {
		p.writeReceipt(id, messageReceiptBlock, msg)
	} else {
		p.writeReceipt(id, messageReceiptDone, nil)
	}
	task.checkMessageTo(msg.to)
}

func (task *ltask) getWorkerId(id serviceId) (workerId int) {
	for i := range task.workers {
		if task.workers[i].running == id {
			return i
		}
	}
	return -1
}

func (task *ltask) initService(L *lua.State, id serviceId, label string,
	source string, chunkName string, workerId int32) (ok bool) {
	ptr := mem.Alloc(uint(unsafe.Sizeof(serviceUd{})))
	ud := (*serviceUd)(unsafe.Pointer(ptr))
	ud.task = task
	ud.id = id
	s := task.services.getService(id)
	if s == nil {
		L.PushString(fmt.Sprintf("Service %d not found", id))
		return
	}
	defer func() {
		if !ok {
			task.services.deleteService(id)
		}
	}()
	if !s.init(task.luaLib, ud, task.services.queueLen, L) || !s.requiref("ltask", ltaskOpen, L) {
		L.PushString(fmt.Sprintf("New service fail: %s", getErrorMessage(L)))
		return
	}
	s.setBinding(workerId)
	if !s.setLabel(label) {
		L.PushString(fmt.Sprintf("Set label fail: %s", getErrorMessage(L)))
		return
	}
	if err := s.loadString(source, chunkName); err != nil {
		L.PushString(fmt.Sprintf("%s", err))
		return
	}
	ok = true
	return
}
