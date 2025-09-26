package ltask

import (
	"go.yuchanns.xyz/lua"
)

func getErrorMessage(L *lua.State) string {
	switch L.Type(-1) {
	case lua.LUA_TLIGHTUSERDATA:
		ptr := (*byte)(L.ToUserData(-1))
		return bytePtrToString(ptr)
	case lua.LUA_TSTRING:
		return L.ToString(-1)
	}
	return "Invalid error message"
}

var pushString = lua.NewCallback(func(L *lua.State) int {
	ptr := (*byte)(L.ToUserData(-1))
	L.SetTop(1)
	L.PushString(bytePtrToString(ptr))
	return 1
})

var requireModule = lua.NewCallback(func(L *lua.State) int {
	name := (*byte)(L.ToUserData(1))
	fn := L.ToUserData(2)
	L.Requiref(bytePtrToString(name), uintptr(fn), false)
	return 0
})
