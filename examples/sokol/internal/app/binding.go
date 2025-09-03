package app

import (
	"unsafe"

	"github.com/smasher164/mem"
	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

func externalOpenLibs(L *lua.State) {
	openLibs := L.FFI().LuaLOpenlibs
	clone := L.Clone
	l := []*lua.Reg{
		{Name: "sapp", Func: openApp},
	}

	L.FFI().LuaLOpenlibs = func(luaL unsafe.Pointer) {
		openLibs(luaL)
		L := clone(luaL)

		L.GetGlobal("package")
		_ = L.GetField(-1, "preload")

		L.SetFuncs(l, 0)
		L.Pop(2)
	}
}

func openApp(L *lua.State) int {
	l := []*lua.Reg{
		{Name: "sendmessage", Func: lsendMessage},
	}
	L.NewLib(l)
	return 1
}

type sokolMessage[T interface{ [2]int | uint64 }] struct {
	Type unsafe.Pointer
	v    T
}

func newMessage(typ unsafe.Pointer, p1, p2 int) (msg *sokolMessage[[2]int]) {
	ptr := mem.Alloc(uint(unsafe.Sizeof(*msg)))
	msg = (*sokolMessage[[2]int])(ptr)
	msg.Type = typ
	msg.v[0] = p1
	msg.v[1] = p2
	return
}

func newMessage64(typ unsafe.Pointer, v uint64) (msg *sokolMessage[uint64]) {
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
	var what unsafe.Pointer
	if L.Type(3) == lua.LUA_TSTRING {
		what = unsafe.Pointer(&[]byte(L.ToString(3))[0])
	} else {
		L.CheckType(3, lua.LUA_TLIGHTUSERDATA)
		what = L.ToPointer(3)
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
