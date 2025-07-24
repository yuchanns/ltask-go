package ltask

import (
	"unsafe"

	"go.yuchanns.xyz/lua"
)

func ltaskInit(L *lua.State) int {
	if L.GetTop() == 0 {
		L.NewTable()
	}
	typ, _ := L.GetField(lua.LUA_REGISTRYINDEX, "LTASK_CONFIG")
	if typ != lua.LUA_TNIL {
		return L.Errorf("Already init")
	}
	L.Pop(1)
	var config *ltaskConfig
	config = (*ltaskConfig)(L.NewUserDataUv(int(unsafe.Sizeof(*config)), 0))
	_ = L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_CONFIG")

	config.Load(L, 1)

	if config.CrashLog[0] != nil {
		// TODO: set crash log
	}

	var task *ltask
	task = (*ltask)(L.NewUserDataUv(int(unsafe.Sizeof(*task)), 0))
	L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_GLOBAL")

	task.config = config
	return 1
}

func ltaskBootstrap(L *lua.State) int {
	l := []luaLReg{
		{"init", ltaskInit},
	}

	luaLNewLib(L, l)
	return 1
}
