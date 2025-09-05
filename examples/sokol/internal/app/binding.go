package app

import (
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/smasher164/mem"
	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/ltask/examples/sokol/internal/sokol"
	"go.yuchanns.xyz/lua"
)

func externalOpenLibs(L *lua.State) {
	ffi := L.Lib().FFI()
	openLibs := ffi.LuaLOpenlibs
	buildState := L.Lib().BuildState
	l := []*lua.Reg{
		{Name: "sapp", Func: openApp},
	}

	ffi.LuaLOpenlibs = func(luaL unsafe.Pointer) {
		openLibs(luaL)
		L := buildState(luaL)

		L.GetGlobal("package")
		_ = L.GetField(-1, "preload")

		L.SetFuncs(l, 0)
		L.Pop(2)
	}
}

func openApp(L *lua.State) int {
	l := []*lua.Reg{
		{Name: "sendmessage", Func: lsendMessage},
		{Name: "unpackmessage", Func: lunpackMessage},
		{Name: "unpackevent", Func: lunpackEvent},
		{Name: "quit", Func: lquit},
	}
	L.NewLib(l)
	return 1
}

type sokolMessage struct {
	Type *byte
	v    uint64
}

func alignUp(n, align int) uint {
	return uint((n + align - 1) &^ (align - 1))
}

func newMessage(typ *byte, p1, p2 uint32) (ptr unsafe.Pointer) {
	ptr = mem.Alloc(alignUp(int(unsafe.Sizeof(sokolMessage{})), 8))
	msg := (*sokolMessage)(ptr)
	msg.Type = typ
	msg.v = uint64(p1)<<32 | uint64(p2)
	return
}

func newMessage64(typ *byte, v uint64) (ptr unsafe.Pointer) {
	ptr = mem.Alloc(alignUp(int(unsafe.Sizeof(sokolMessage{})), 8))
	msg := (*sokolMessage)(ptr)
	msg.Type = typ
	msg.v = v
	return
}

func lsendMessage(L *lua.State) int {
	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	L.CheckType(2, lua.LUA_TLIGHTUSERDATA)
	sendMessage := *(*ltask.ExternalSend)(L.ToPointer(1))
	p := L.ToPointer(2)
	L.CheckType(3, lua.LUA_TSTRING)
	what := L.Lib().FFI().LuaTolstring(L.L(), 3, nil)
	p1 := L.OptInteger(4, 0)
	var msg unsafe.Pointer
	if L.GetTop() < 5 || L.IsNoneOrNil(5) {
		msg = newMessage64(what, uint64(p1))
	} else {
		p2 := L.CheckInteger(5)
		msg = newMessage(what, uint32(p1), uint32(p2))
	}
	sendMessage(p, msg)
	return 0
}

func lunpackMessage(L *lua.State) int {
	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	m := (*sokolMessage)(L.ToPointer(1))
	L.PushString(bytePtrToString(m.Type))
	L.PushInteger(int64(uint32(m.v)))
	L.PushInteger(int64(m.v >> 32))
	L.PushInteger(int64(m.v))
	m.Type = nil
	mem.Free(unsafe.Pointer(m))
	return 4
}

func lunpackEvent(L *lua.State) int {
	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	ev := (*sokol.SappEvent)(L.ToPointer(1))
	em := eventUnpack(ev)
	L.PushString(em.typ)
	L.PushInteger(int64(em.p1))
	L.PushInteger(int64(em.p2))
	return 3
}

func lquit(L *lua.State) int {
	var quit func()
	purego.RegisterLibFunc(&quit, L.Lib().FFI().Lib(), "sapp_request_quit")
	quit()
	return 0
}
