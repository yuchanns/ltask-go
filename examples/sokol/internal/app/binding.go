package app

import (
	"unsafe"

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
	}
	L.NewLib(l)
	return 1
}

type sokolMessage[T interface{ [2]int | uint64 }] struct {
	Type string
	v    T
}

func newMessage(typ string, p1, p2 int) (msg *sokolMessage[[2]int]) {
	ptr := mem.Alloc(uint(unsafe.Sizeof(*msg)))
	msg = (*sokolMessage[[2]int])(ptr)
	msg.Type = typ
	msg.v[0] = p1
	msg.v[1] = p2
	return
}

func newMessage64(typ string, v uint64) (msg *sokolMessage[uint64]) {
	ptr := mem.Alloc(uint(unsafe.Sizeof(*msg)))
	msg = (*sokolMessage[uint64])(ptr)
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
		msg = unsafe.Pointer(newMessage(what, int(p1), int(p2)))
	}
	sendMessage(p, msg)
	return 0
}

func lunpackMessage(L *lua.State) int {
	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	m := (*sokolMessage[[2]int])(L.ToPointer(1))
	L.PushString(m.Type)
	L.PushInteger(int64(m.v[0]))
	L.PushInteger(int64(m.v[1]))
	var u64 uint64
	if unsafe.Sizeof(m.v) >= 8 {
		u64 = *(*uint64)(unsafe.Pointer(&m.v))
	}
	L.PushInteger(int64(u64))
	mem.Free(unsafe.Pointer(m))
	return 4
}
