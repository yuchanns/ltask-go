package ltask

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/phuslu/log"
	"go.yuchanns.xyz/xxchan"
	"go.yuchanns.xyz/xxcond"
)

const bindingServiceQueue = 16

type serviceId = int32

type workerThread struct {
	task         *ltask
	workerId     int32
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
	atomic.StoreInt32(&w.serviceReady, 0)
	atomic.StoreInt32(&w.serviceDone, 0)
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
		atomic.CompareAndSwapInt32(&w.serviceReady, job, 0)
		id = serviceId(job)
		break
	}
	return
}

func (w *workerThread) acquireScheduler() (ok bool) {
	ok = atomic.CompareAndSwapInt32(&w.task.scheduleOwner, threadNone, w.workerId)
	if ok {
		log.Debug().Msgf("Worker %d acquired scheduler", w.workerId)
	}

	return
}

func (w *workerThread) releaseScheduler() {
	if atomic.LoadInt32(&w.task.scheduleOwner) != w.workerId {
		panic("Worker trying to release scheduler it does not own")
	}
	atomic.CompareAndSwapInt32(&w.task.scheduleOwner, w.workerId, threadNone)
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
	if atomic.CompareAndSwapInt32(&w.serviceDone, 0, w.running) {
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
	atomic.StoreInt32(&w.serviceReady, job)
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
	if atomic.CompareAndSwapInt32(&w.serviceReady, job, 0) {
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
	atomic.AddInt32(&w.task.activeWorker, 1)
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
				atomic.AddInt32(&w.task.threadCount, -1)
				log.Debug().Msgf("Worker %d sleeping", w.workerId)
				w.sleep()
				atomic.AddInt32(&w.task.activeWorker, 1)
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
				panic(fmt.Sprintf("Service %d not in schedule status: %d", id, status))
			}
			p.setStatus(id, serviceStatusRunning)
			if !p.resume(id) {
				dead = true
				log.Debug().Msgf("Service %d quit", id)
				p.setStatus(id, serviceStatusDead)
				if id == serviceIdRoot {
					log.Debug().Msg("Root quit")
					// root quit, wakeup others
					w.task.quitAllWorkers()
					w.task.triggerAllSockevent()
					w.task.wakeupAllWorkers()
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
	atomic.AddInt32(&w.task.threadCount, -1)
	log.Debug().Msgf("Worker %d quit", w.workerId)

}
