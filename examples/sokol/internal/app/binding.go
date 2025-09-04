package app

import (
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/smasher164/mem"
	"go.yuchanns.xyz/ltask"
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
		{Name: "quit", Func: lquit},
	}
	L.NewLib(l)
	return 1
}

type sokolMessage struct {
	Type string
	v    uint64
}

func alignUp(n, align int) uint {
	return uint((n + align - 1) &^ (align - 1))
}

func newMessage(typ string, p1, p2 uint32) (msg *sokolMessage) {
	ptr := mem.Alloc(alignUp(int(unsafe.Sizeof(*msg)), 8))
	msg = (*sokolMessage)(ptr)
	msg.Type = typ
	msg.v = uint64(p1)<<32 | uint64(p2)
	return
}

func newMessage64(typ string, v uint64) (msg *sokolMessage) {
	ptr := mem.Alloc(alignUp(int(unsafe.Sizeof(*msg)), 8))
	msg = (*sokolMessage)(ptr)
	msg.Type = typ
	msg.v = v
	return
}

func lsendMessage(L *lua.State) int {
	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	L.CheckType(2, lua.LUA_TLIGHTUSERDATA)
	sendMessage := *(*ltask.ExternalSend)(L.ToPointer(1))
	p := L.ToPointer(2)
	var what string
	if L.Type(3) == lua.LUA_TSTRING {
		what = L.ToString(3)
	} else {
		L.CheckType(3, lua.LUA_TLIGHTUSERDATA)
		// TODO: what = L.ToPointer(3)
	}
	p1 := L.OptInteger(4, 0)
	var msg unsafe.Pointer
	if L.GetTop() < 5 || L.IsNoneOrNil(5) {
		msg = unsafe.Pointer(newMessage64(what, uint64(p1)))
	} else {
		p2 := L.CheckInteger(5)
		msg = unsafe.Pointer(newMessage(what, uint32(p1), uint32(p2)))
	}
	sendMessage(p, msg)
	return 0
}

func lunpackMessage(L *lua.State) int {
	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	m := (*sokolMessage)(L.ToPointer(1))
	L.PushString(m.Type)
	L.PushInteger(int64(uint32(m.v)))
	L.PushInteger(int64(m.v >> 32))
	L.PushInteger(int64(m.v))
	mem.Free(unsafe.Pointer(m))
	return 4
}

func lquit(L *lua.State) int {
	var quit func()
	purego.RegisterLibFunc(&quit, L.Lib().FFI().Lib(), "sapp_request_quit")
	quit()
	return 0
}
