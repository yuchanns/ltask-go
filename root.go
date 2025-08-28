package ltask

import (
	"sync/atomic"

	"go.yuchanns.xyz/lua"
)

func ltaskInitService(L *lua.State) int {
	s := getS(L)
	sid := L.CheckInteger(1)
	label := L.CheckString(2)
	source := L.CheckString(3)
	chunkName := L.CheckString(4)
	workerId := int32(L.OptInteger(5, -1))

	if !s.task.initService(L, serviceId(sid), label, source, chunkName, workerId) {
		L.PushBoolean(false)
		L.Insert(-2)
		return 2
	}

	L.PushBoolean(true)
	return 1
}

func ltaskCloseService(L *lua.State) int {
	s := getS(L)
	id := serviceId(L.CheckInteger(1))
	if s.task.services.getStatus(id) != serviceStatusDead {
		return L.Errorf("Hang %d before close it", id)
	}
	sockId := s.task.services.getSockevent(id)
	if sockId >= 0 {
		s.task.event[sockId].close()
		atomic.StoreInt32(&s.task.eventInit[sockId], 0)
	}
	ret := s.task.services.closeServiceMessages(L, id)
	s.task.services.deleteService(id)
	return ret
}
