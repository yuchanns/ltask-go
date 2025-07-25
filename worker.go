package ltask

import "sync"

const bindingServiceQueue = 16

type serviceId uint64

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
