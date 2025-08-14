package ltask

import (
	"bytes"
	"fmt"
	"slices"
	"time"
	"unsafe"

	"github.com/phuslu/log"
	"github.com/smasher164/mem"
	"go.yuchanns.xyz/lua"
	"go.yuchanns.xyz/xxchan"
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

const (
	serviceIdSystem = 0
	serviceIdRoot   = 1
)

type memoryStat struct {
	count [typeIdCount]int64
	mem   int64
	limit int64
}

type service struct {
	L             *lua.State
	rL            *lua.State
	msg           *xxchan.Channel[*message]
	out           *message
	bounce        *message
	status        int64
	receipt       int64
	bindingThread int32
	sockeventId   int64
	id            serviceId
	label         [32]byte
	stat          memoryStat
	cpucost       uint64
	clock         uint64
}

func (s *service) init(luaLib *lua.Lib, ud *serviceUd, queueLen int64, pL *lua.State) (ok bool) {
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
	ptr := malloc.Alloc(uint(xxchan.Sizeof[*message](int(queueLen))))
	s.msg = xxchan.Make[*message](ptr, int(queueLen))
	s.L = L
	s.rL = L.NewThread()
	L.Ref(lua.LUA_REGISTRYINDEX)
	return true
}

func (s *service) requiref(name string, fn lua.GoFunc, pL *lua.State) (ok bool) {
	if s.rL == nil {
		errorMessage(nil, pL, "requiref: No service")
		return false
	}
	L := s.rL
	L.PushGoFunction(requireModule)
	L.PushLightUserData(&name)
	L.PushLightUserData(&fn)
	if L.PCall(2, 0, 0) != nil {
		errorMessage(L, pL, "requiref: pcall error")
		L.Pop(1)
		return false
	}
	return true
}

func (s *service) setBinding(workerThread int32) {
	s.bindingThread = workerThread
}

func (s *service) setLabel(label string) (ok bool) {
	if len(label) > 32 {
		label = label[:32]
	}
	copy(s.label[:], label)
	return true
}

func (s *service) loadString(source string, chunkName string) (err error) {
	if s.L == nil {
		err = fmt.Errorf("Init service first")
		return
	}
	L := s.L
	if err = L.LoadBuffer([]byte(source), chunkName); err != nil {
		s.status = serviceStatusDead
		return
	}
	s.status = serviceStatusIdle
	return
}

func (s *service) close() {
	if s == nil {
		return
	}
	if s.L != nil {
		s.L.Close()
	}
	if s.msg != nil {
		for s.msg.Len() > 0 {
			msg, ok := s.msg.Pop()
			if ok {
				msg.delete()
			}
		}
		malloc.Free(unsafe.Pointer(s.msg))
		s.msg = nil
	}
	s.out = nil
	s.bounce = nil
}

type servicePool struct {
	mask     int32
	queueLen int64
	id       int32
	s        []*service
}

func newServicePool(config *ltaskConfig) (pool *servicePool) {
	structSize := int(unsafe.Sizeof(servicePool{}))
	elemSize := int(unsafe.Sizeof(&service{}))
	elemAlign := int(unsafe.Alignof(&service{}))
	ptr := malloc.Alloc(uint(alignUp(structSize+elemSize*int(config.maxService), elemAlign)))
	pool = (*servicePool)(unsafe.Pointer(ptr))
	pool.mask = config.maxService - 1
	pool.id = 0
	pool.queueLen = config.queueSending
	pool.s = unsafe.Slice((**service)(unsafe.Pointer(uintptr(ptr)+uintptr(structSize))), int(config.maxService))
	return
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

func (p *servicePool) initSockevent(id serviceId, index int64) {
	s := p.getService(id)
	if s == nil {
		return
	}
	s.sockeventId = index
}

func (p *servicePool) getSockevent(id serviceId) (sockeventId int64) {
	s := p.getService(id)
	if s == nil {
		return
	}
	sockeventId = s.sockeventId
	return
}

func (p *servicePool) sendSignal(id serviceId) {
	s := p.getService(id)
	if s == nil {
		return
	}
	if s.out != nil {
		s.out.delete()
	}
	s.out = newMessage(&message{
		from:    id,
		to:      serviceIdRoot,
		session: 0,
		typ:     messageTypeSignal,
	})
}

func (p *servicePool) readReceipt(id serviceId) (receipt int64, r *message) {
	s := p.getService(id)
	if s == nil {
		receipt = messageReceiptNone
		return
	}
	receipt = s.receipt
	r = s.bounce
	s.receipt = messageReceiptNone
	s.bounce = nil
	return
}

func (p *servicePool) sendMessage(id serviceId, msg *message) (ok bool) {
	s := p.getService(id)
	if s == nil || s.out != nil {
		return
	}
	s.out = msg
	ok = true
	return
}

func (p *servicePool) outMessage(id serviceId) (out *message) {
	s := p.getService(id)
	if s == nil {
		return
	}
	out = s.out
	s.out = nil

	return
}

func (p *servicePool) getLabel(id serviceId) string {
	s := p.getService(id)
	if s == nil {
		return "<dead service>"
	}
	n := bytes.IndexByte(s.label[:], 0)
	if n == -1 {
		n = len(s.label)
	}
	return string(s.label[:n])
}

func (p *servicePool) writeReceipt(id serviceId, receipt int64, bounce *message) {
	s := p.getService(id)
	if s == nil || s.receipt != messageReceiptNone {
		log.Error().Msgf("WARNING: write recipt %d fail (%d)", id, s.receipt)
	}
	if s != nil {
		s.bounce.delete()
		s.receipt = receipt
		s.bounce = bounce
	}
}

func (p *servicePool) popMessage(id serviceId) (msg *message) {
	s := p.getService(id)
	if s == nil {
		return
	}
	if s.bounce != nil {
		msg = s.bounce
		s.bounce = nil
		return
	}
	msg, _ = s.msg.Pop()
	return
}

func (p *servicePool) hasMessage(id serviceId) (has bool) {
	s := p.getService(id)
	if s == nil {
		return
	}
	if s.receipt != messageReceiptNone {
		has = true
		return
	}
	return s.msg.Len() > 0
}

func (p *servicePool) pushMessage(id serviceId, msg *message) (block bool) {
	s := p.getService(id)
	if s == nil || s.status == serviceStatusDead {
		return
	}
	block = !s.msg.Push(msg)
	return
}

func (p *servicePool) resume(id serviceId) (yield bool) {
	s := p.getService(id)
	if s == nil {
		return
	}
	L := s.L
	if L == nil {
		return
	}
	start := time.Now()
	nres, yield, err := L.Resume(nil, 0)
	s.cpucost = uint64(time.Since(start).Nanoseconds())
	if err == nil {
		if yield {
			L.Pop(int(nres))
		}
		return
	}
	if !L.CheckStack(lua.LUA_MINSTACK) {
		log.Error().Msgf("%s", err)
		return
	}
	L.PushString(fmt.Sprintf("Service %d error: %s", id, err))
	L.Traceback(L, err.Error(), 0)
	log.Error().Msgf("%s", err)
	L.Pop(2)
	return
}

func (p *servicePool) setStatus(id serviceId, status int64) {
	s := p.getService(id)
	if s == nil {
		return
	}
	log.Debug().Msgf("Set service %d status to %d", id, status)
	s.status = status
}

func (p *servicePool) getStatus(id serviceId) (status int64) {
	s := p.getService(id)
	if s == nil {
		status = serviceStatusDead
		return
	}
	status = s.status
	return
}

func (p *servicePool) postMessage(msg *message) (ok bool) {
	s := p.getService(msg.to)
	if s == nil || s.status == serviceStatusDead {
		return
	}
	ok = s.msg.Push(msg)
	return
}

func (p *servicePool) newService(sid serviceId) (svcId serviceId) {
	var id serviceId
	if sid != 0 {
		id = sid
		if p.getService(sid) != nil {
			return
		}
	} else {
		id = p.id
		for i := int32(0); ; i++ {
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
	var s *service
	ptr := malloc.Alloc(uint(unsafe.Sizeof(*s)))
	s = (*service)(unsafe.Pointer(ptr))
	s.receipt = messageReceiptNone
	s.id = svcId
	s.msg = nil
	s.out = nil
	s.bounce = nil
	s.status = serviceStatusUninitialized
	s.bindingThread = -1
	s.cpucost = 0
	s.clock = 0
	s.sockeventId = -1
	s.label = [32]byte{}
	p.setService(s)
	return
}

func (p *servicePool) setBindingThread(id serviceId, thread int32) {
	s := p.getService(id)
	if s == nil {
		return
	}
	s.bindingThread = thread
}

func (p *servicePool) getBindingThread(id serviceId) (thread int32) {
	s := p.getService(id)
	if s == nil {
		return
	}
	thread = s.bindingThread
	return
}

func (p *servicePool) getService(id int32) *service {
	return p.s[id&p.mask]
}

func (p *servicePool) setService(svc *service) {
	p.s[svc.id&p.mask] = svc
}

func (p *servicePool) destroy() {
	if p == nil {
		return
	}
	for i := range p.s {
		s := p.s[i]
		if s == nil {
			continue
		}
		s.close()
		malloc.Free(unsafe.Pointer(s))
	}
	malloc.Free(unsafe.Pointer(p))
}

// OnServiceInit registers lua.GoFunc to be called when a service is initialized.
// This allows users to add custom Lua functions or libraries written in Go that will be available
// in each service's Lua state.
// It is not thread-safe and should be called before ltask.OpenLibs.
//
// Example:
//
//	OnServiceInit(func(L *lua.State) int {
//		L.Requiref("my_module", myModuleOpen, false)
//		return 0
//	})
//	OpenLibs(L)
//	L.DoFile("my_script.lua")
func OnServiceInit(f ...lua.GoFunc) {
	luaOpenFuncs = append(luaOpenFuncs, f...)
}

var luaOpenFuncs = []lua.GoFunc{}

func initService(L *lua.State) int {
	// TODO: set ud as a string
	ud := L.ToUserData(1)
	L.PushLightUserData(ud)
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_ID")
	L.OpenLibs()
	for _, lOpen := range luaOpenFuncs {
		L.PushGoFunction(lOpen)
		err := L.PCall(0, 0, 0)
		if err != nil {
			return L.Errorf("Init service error: %s", err)
		}
	}
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

func (p *servicePool) closeServiceMessages(L *lua.State, id serviceId) (reportError int) {
	var index int
	for {
		m := p.popMessage(id)
		if m == nil {
			break
		}
		if slices.Contains([]int{messageTypeRequest, messageTypeSystem}, m.typ) {
			if reportError == 0 {
				L.NewTable()
				reportError = 1
				index = 1
			}
			L.PushInteger(int64(m.from))
			L.RawSetI(-2, int64(index))
			index++
			L.PushInteger(int64(m.session))
			L.RawSetI(-2, int64(index))
			index++
		}
		m.delete()
	}
	return
}

func (p *servicePool) deleteService(id serviceId) {
	s := p.getService(id)
	if s == nil {
		return
	}
	s.close()
	malloc.Free(unsafe.Pointer(s))
	p.s[id&p.mask] = nil
}

func requireModule(L *lua.State) int {
	name := *(*string)(L.ToUserData(1))
	fn := *(*lua.GoFunc)(L.ToUserData(2))
	L.Requiref(name, fn, false)
	return 0
}

func (task *ltask) initService(L *lua.State, id serviceId, label string,
	source string, chunkName string, workerId int32) (ok bool) {
	// FIXME: free memory of ud when service is delete
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
