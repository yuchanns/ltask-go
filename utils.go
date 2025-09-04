package ltask

import (
	"go.yuchanns.xyz/lua"
)

func getErrorMessage(L *lua.State) string {
	switch L.Type(-1) {
	case lua.LUA_TLIGHTUSERDATA:
		return *(*string)(L.ToUserData(-1))
	case lua.LUA_TSTRING:
		return L.ToString(-1)
	}
	return "Invalid error message"
}

func pushString(L *lua.State) int {
	msg := *(*string)(L.ToUserData(-1))
	L.SetTop(1)
	L.PushString(msg)
	return 1
}

func requireModule(L *lua.State) int {
	name := *(*string)(L.ToUserData(1))
	fn := *(*lua.GoFunc)(L.ToUserData(2))
	L.Requiref(name, fn, false)
	return 0
}
