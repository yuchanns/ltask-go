package ltask

import (
	"unsafe"

	"go.yuchanns.xyz/lua"
)

type session = uint64

type message struct {
	from    serviceId
	to      serviceId
	session session
	typ     int
	msg     unsafe.Pointer
	sz      int64
}

func genSendMessage(L *lua.State, id serviceId) *message {
	m := &message{
		from:    id,
		to:      L.CheckInteger(1),
		session: session(L.CheckInteger(2)),
		typ:     int(L.CheckInteger(3)),
	}
	if !L.IsNoneOrNil(4) {
		L.CheckType(4, lua.LUA_TLIGHTUSERDATA)
		m.msg = L.ToUserData(4)
		m.sz = L.CheckInteger(5)
	}
	return newMessage(m)
}

func newMessage(m *message) *message {
	ptr := malloc.Alloc(uint(unsafe.Sizeof(*m)))
	msg := (*message)(unsafe.Pointer(ptr))
	*msg = *m
	return msg
}

func (m *message) delete() {
	if m == nil {
		return
	}
	malloc.Free(unsafe.Pointer(m))
}

const (
	messageReceiptNone     = 0
	messageReceiptDone     = 1
	messageReceiptError    = 2
	messageReceiptBlock    = 3
	messageReceiptResponse = 4
)

const (
	messageTypeSystem   = 0
	messageTypeRequest  = 1
	messageTypeResponse = 2
	messageTypeError    = 3
	messageTypeSignal   = 4
	messageTypeIdle     = 5
)

const (
	messageScheduleNew = 0
	messageScheduleDel = 1
)
