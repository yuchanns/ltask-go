package ltask

import (
	"sync/atomic"
	"unsafe"

	"go.yuchanns.xyz/lua"
)

var malloc *arena

func ltaskInit(L *lua.State) int {
	if L.GetTop() == 0 {
		L.NewTable()
	}
	typ, _ := L.GetField(lua.LUA_REGISTRYINDEX, "LTASK_CONFIG")
	if typ != lua.LUA_TNIL {
		return L.Errorf("Already init")
	}
	L.Pop(1)

	malloc = createArena(1024 * 1024)

	var config *ltaskConfig
	config = (*ltaskConfig)(L.NewUserDataUv(int(unsafe.Sizeof(*config)), 0))
	_ = L.SetField(lua.LUA_REGISTRYINDEX, "LTASK_CONFIG")

	config.load(L, 1)

	if config.crashLog[0] != nil {
		// TODO: set crash log
	}

	var task *ltask
	task.init(L, config)

	return 1
}

var bootInit atomic.Int32

func ltaskBootstrap(L *lua.State) int {
	if bootInit.Add(1) != 1 {
		return L.Errorf("ltask.bootstrap can only require once")
	}
	l := []luaLReg{
		{"init", ltaskInit},
	}

	luaLNewLib(L, l)
	return 1
}
