package ltask

import (
	"fmt"
	"unsafe"

	"go.yuchanns.xyz/lua"
)

const (
	typeIdCount = 6
)

const (
	serviceStatusUninitialized = 0
	serviceStatusIdle          = 1
	serviceStatusSchedule      = 2
	serviceStatusRunning       = 3
	serviceStatusDone          = 4
	serviceStatusDead          = 5
)

type memoryStat struct {
	count [typeIdCount]int64
	mem   int64
	limit int64
}

type service struct {
	L             *lua.State
	rL            *lua.State
	msg           chan *message
	out           *message
	bounce        *message
	status        int64
	receipt       int64
	bindingThread int64
	id            serviceId
	label         [32]byte
	stat          memoryStat
	cpucost       uint64
	clock         uint64
}

func (s *service) close() {
	if s.L != nil {
		s.L.Close()
	}
	if s.msg != nil {
		close(s.msg)
		for range s.msg {
		}
		s.msg = nil
	}
	s.out = nil
	s.bounce = nil
}

type servicePool struct {
	mask     int64
	queueLen int64
	id       int64
	s        []*service
}

func newServicePool(config *ltaskConfig) (pool *servicePool) {
	pool = &servicePool{
		mask:     config.maxService - 1,
		queueLen: config.queueSending,
		id:       typeIdCount,
		s:        make([]*service, config.maxService),
	}
	return
}

func (p *servicePool) newService(sid int64) (svcId serviceId) {
	var id int64
	if sid != 0 {
		id = sid
		if p.getService(sid) != nil {
			return
		}
	} else {
		id = p.id
		for i := int64(0); ; i++ {
			if i > p.mask {
				return
			}
			id++
			if p.getService(id) == nil {
				break
			}
		}
		p.id = id + 1
	}
	svcId = id
	s := &service{
		receipt:       messageReceiptNone,
		id:            id,
		status:        serviceStatusUninitialized,
		bindingThread: -1,
		cpucost:       0,
		clock:         0,
	}
	p.setService(s)
	return
}

func (p *servicePool) getService(id int64) *service {
	return p.s[id&p.mask]
}

func (p *servicePool) setService(svc *service) {
	p.s[svc.id&p.mask] = svc
}

func initService(L *lua.State) int {
	// ud := (*byte)(L.ToUserData(1))
	// size := L.ToInteger(2)
	// initServiceKey
	L.OpenLibs()
	return 0
}

func pushString(L *lua.State) int {
	msg := *(*string)(L.ToUserData(-1))
	L.SetTop(1)
	L.PushString(msg)
	return 1
}

func errorMessage(fromL, toL *lua.State, msg string) {
	if toL == nil {
		return
	}
	if fromL != nil {
		errMsg := fromL.ToString(-1)
		toL.PushGoFunction(pushString)
		toL.PushLightUserData(unsafe.Pointer(&errMsg))
		if toL.PCall(1, 1, 0) == nil {
			return
		}
		toL.Pop(1)
	}
	toL.PushLightUserData(unsafe.Pointer(&msg))
}

func (p *servicePool) initService(ud *serviceUd, pL *lua.State) (ok bool) {
	s := p.getService(ud.id)
	// TODO: compatible 505
	// malloc
	L, err := luaLib.NewState()
	if err != nil {
		return
	}
	L.PushGoFunction(initService)
	L.PushLightUserData(ud)
	L.PushInteger(int64(unsafe.Sizeof(*ud)))
	if err := L.PCall(2, 0, 0); err != nil {
		errorMessage(L, pL, "Init lua state error")
		L.Close()
		return
	}
	s.msg = make(chan *message, p.queueLen)
	s.L = L
	s.rL = L.NewThread()
	return true
}

func (p *servicePool) deleteService(id serviceId) {
	s := p.getService(id)
	if s == nil {
		return
	}
	p.setService(nil)
	s.close()
}

func (task *ltask) newService(L *lua.State, id serviceId, label string,
	source string, sourceSz int, chunkName string, workerId int64) (ok bool) {
	ud := &serviceUd{
		task: task,
		id:   id,
	}
	if !task.services.initService(ud, L) {
		task.services.deleteService(id)
		L.PushString(fmt.Sprintf("New service fail: %s", getErrorMessage(L)))
		return false
	}
	return true
}
