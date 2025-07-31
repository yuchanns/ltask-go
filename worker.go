package ltask

import (
	"sync"
	"sync/atomic"

	"github.com/phuslu/log"
)

const bindingServiceQueue = 16

type serviceId = int64

type bindingService struct {
	head int64
	tail int64
	q    [bindingServiceQueue]serviceId
}

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
	trigger      *sync.Cond
}

func (w *workerThread) destroy() {
	w.trigger = nil
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
	// jobs := w.task.getPendingJobs()

	// 4. assign queue task

	// 5. assign task to workers
}

func (task *ltask) dispatchExternalMessages() {
	var send bool
	if task.externalLastMessage != nil {
		if task.services.pushMessage(task.externalLastMessage) {
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
		m := &message{
			from:    0,
			to:      1, // root
			session: 0,
			typ:     messageTypeRequest,
			msg:     msg,
			sz:      int64(sz),
		}
		if task.services.pushMessage(m) {
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
			if p.pushMessage(msg) {
				log.Debug().Msgf("Root service is blocked, service %d will try to signal it later", id)
				task.scheduleBack(id)
				continue
			}
			log.Debug().Msgf("Service %d signaled root service", id)
			task.checkMessageTo(msg.to)
			continue
		}
		if msg != nil {
			task.dispatchOutMessage(msg)
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

func (task *ltask) dispatchOutMessage(msg *message) {
	p := task.services
	if msg.to == serviceIdSystem {
		return
	}
	if s := p.getService(msg.to); s == nil || s.status == serviceStatusDead {
		p.writeReceipt(msg.to, messageReceiptError, msg)
	} else if p.pushMessage(msg) {
		p.writeReceipt(msg.to, messageReceiptBlock, msg)
	} else {
		p.writeReceipt(msg.to, messageReceiptDone, nil)
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

func (w *workerThread) sleep() {
	if w.termSignal > 0 {
		return
	}
	// FIXME: currently we set to term once sleep
	w.termSignal = 1
	return

	w.trigger.L.Lock()
	defer w.trigger.L.Unlock()

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

func (w *workerThread) hasJob() bool {
	return w.serviceReady != 0
}

func (w *workerThread) quit() {
	w.trigger.L.Lock()
	defer w.trigger.L.Unlock()

	w.trigger.Signal()

	w.sleeping = 0
}
