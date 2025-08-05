package ltask

import (
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/phuslu/log"
	"go.yuchanns.xyz/xxchan"
	"go.yuchanns.xyz/xxcond"
)

const bindingServiceQueue = 16

type serviceId = int64

type workerThread struct {
	task         *ltask
	workerId     int64
	running      serviceId
	binding      serviceId
	waiting      serviceId
	serviceReady atomicInt
	serviceDone  atomicInt
	termSignal   int64
	sleeping     int64
	wakeup       int64
	busy         int64
	trigger      *xxcond.Cond
	bindingQueue *xxchan.Channel[serviceId]
	scheduleTime int64
}

func (w *workerThread) init(task *ltask, id serviceId) {
	w.task = task
	w.workerId = id
	w.running = 0
	w.binding = 0
	w.waiting = 0
	atomic.StoreInt64(&w.serviceReady, 0)
	atomic.StoreInt64(&w.serviceDone, 0)
	w.termSignal = 0
	w.sleeping = 0
	w.wakeup = 0
	w.busy = 0

	ptr := malloc.Alloc(uint(xxcond.Sizeof()))
	w.trigger = xxcond.Make(ptr)

	ptr = malloc.Alloc(uint(xxchan.Sizeof[serviceId](bindingServiceQueue)))
	w.bindingQueue = xxchan.Make[serviceId](ptr, bindingServiceQueue)
}

func (w *workerThread) destroy() {
	malloc.Free(unsafe.Pointer(w.trigger))
	malloc.Free(unsafe.Pointer(w.bindingQueue))
}

// getJob retrieves a job from the worker's serviceReady queue.
func (w *workerThread) getJob() (id serviceId) {
	for {
		job := w.serviceReady
		if job == 0 {
			break
		}
		atomic.CompareAndSwapInt64(&w.serviceReady, job, 0)
		id = serviceId(job)
		break
	}
	return
}

func (w *workerThread) acquireScheduler() (ok bool) {
	ok = atomic.CompareAndSwapInt64(&w.task.scheduleOwner, threadNone, w.workerId)
	if ok {
		log.Debug().Msgf("Worker %d acquired scheduler", w.workerId)
	}

	return
}

func (w *workerThread) releaseScheduler() {
	if atomic.LoadInt64(&w.task.scheduleOwner) != w.workerId {
		panic("Worker trying to release scheduler it does not own")
	}
	atomic.CompareAndSwapInt64(&w.task.scheduleOwner, w.workerId, threadNone)
	log.Debug().Msgf("Worker %d released scheduler", w.workerId)
}

func (w *workerThread) dispatch() {
	// 0: dispatch external messages
	if w.task.externalMessage != nil {
		w.task.dispatchExternalMessages()
	}
	// 1: collect done services
	doneJobs := w.task.collectDoneJobs()

	// 2: dispatch out message by doneJobs
	w.task.dispatchOutMessages(doneJobs)

	// 3: get pending jobs
	jobs := w.task.getPendingJobs()

	// 4. assign queue task
	freeSlots := w.task.countFreeSlots()

	if freeSlots < len(jobs) {
		panic("Not enough free slots to assign jobs")
	}

	// 5. assign task to workers
	jobs = w.task.prepare(jobs, freeSlots-len(jobs))

	// 6. assign prepared tasks
	w.task.assignPrepare(jobs)

	// 7
	w.task.triggerBlockedWorkers()
}

func (task *ltask) wakeupAlWorkers() {
	for i := range task.workers {
		task.workers[i].wake()
	}
}

func (task *ltask) quitAllWorkers() {
	for i := range task.workers {
		task.workers[i].termSignal = 1
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
			to:      1, // root
			session: 0,
			typ:     messageTypeRequest,
			msg:     msg,
			sz:      int64(sz),
		})
		if task.services.pushMessage(m.to, m) {
			// blocked, save the message for next time
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
			// continue waiting for blocked service running
			blocked = 1
			continue
		}
		// TODO: touch service who block the waiting service
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
				// assign a none-binding service
				break
			}
		}
	}
}

func (task *ltask) prepare(prepare []serviceId, freeSlots int) []serviceId {
	for range freeSlots {
		job, ok := task.schedule.Pop()
		if !ok {
			// no more job
			break
		}
		id := serviceId(job)
		worker := task.services.getBindingThread(id)
		if worker < 0 {
			// no binding worker
			prepare = append(prepare, id)
			continue
		}
		w := &task.workers[worker]
		if !w.bindingQueue.Push(id) {
			// binding queue is full, we can't bind this service
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
		atomic.StoreInt64(&w.serviceReady, id)
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
			// TODO: schedule back for sockevent
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
		// only root can send schedule message
		p.writeReceipt(id, messageReceiptError, msg)
		return
	}
	sid := msg.session
	switch msg.typ {
	case messageScheduleNew:
		msg.to = p.newService(int64(sid))
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

func (w *workerThread) doneJob() (job serviceId) {
	done := w.serviceDone
	// We do not need CAS here as it is guaranteed that
	// only one worker acquires the scheduler at a time.
	if done > 0 {
		w.serviceDone = 0
	}
	job = serviceId(done)
	return
}

func (w *workerThread) completeJob() (ok bool) {
	if atomic.CompareAndSwapInt64(&w.serviceDone, 0, w.running) {
		w.running = 0
		ok = true
	}
	return
}

func (w *workerThread) schedule() (noJob bool) {
	w.dispatch()
	if w.hasJob() {
		return
	}
	if w.binding > 0 {
		// binding a service and no job to do
		noJob = true
		return
	}

	// Try to steal a job from other workers
	job := w.stealJob()
	if job == 0 {
		noJob = true
		return
	}
	log.Debug().Msgf("Worker %d stealing service %d", w.workerId, job)
	atomic.StoreInt64(&w.serviceReady, job)
	return
}

func (w *workerThread) stealJob() (job serviceId) {
	for i := range w.task.workers {
		job = w.task.workers[i].stolen()
		if job != 0 {
			break
		}
	}
	return
}

func (w *workerThread) stolen() (id serviceId) {
	job := w.serviceReady
	if job == 0 {
		return
	}
	workerId := w.task.services.getBindingThread(job)
	if w.workerId == workerId {
		// job is binding to the worker, can't steal
		return
	}
	if atomic.CompareAndSwapInt64(&w.serviceReady, job, 0) {
		id = job
		w.waiting = 0
	}
	return
}

func (w *workerThread) assignJob(id serviceId) (ret serviceId) {
	if w.serviceReady != 0 {
		// There is already a job assigned, can't assign another one
		return
	}
	if job, ok := w.bindingQueue.Pop(); ok {
		id = job
	}
	w.serviceReady = id
	ret = id
	return
}

func (w *workerThread) sleep() {
	if w.termSignal > 0 {
		return
	}

	w.trigger.Lock()
	defer w.trigger.Unlock()

	if w.hasJob() {
		w.wakeup = 0
		return
	}
	if w.wakeup != 0 {
		w.wakeup = 0
		return
	}
	w.sleeping = 1
	w.trigger.Wait()
	w.sleeping = 0
}

func (w *workerThread) kickRunning(id serviceId) {
	w.task.blockedService = 1
	w.waiting = id
}

func (w *workerThread) wake() {
	if w.sleeping == 0 {
		return
	}
	w.wakeup = 1
	w.trigger.Signal()
}

func (w *workerThread) hasJob() bool {
	return w.serviceReady != 0
}

func (w *workerThread) quit() {
	w.trigger.Signal()

	w.sleeping = 0
}

func (w *workerThread) start() {
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
					runtime.Gosched()
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
				w.task.services.sendSignal(id)
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
					runtime.Gosched()
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
