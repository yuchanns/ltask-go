package ltask

import (
	"unsafe"

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
