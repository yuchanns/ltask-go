package ltask

import (
	"sync/atomic"
	"unsafe"

	"go.yuchanns.xyz/lua"
)

func getPtr[T any](L *lua.State, key string) *T {
	typ, _ := L.GetField(lua.LUA_REGISTRYINDEX, key)
	if typ == lua.LUA_TNIL {
		L.Errorf("%s is absense", key)
		return nil
	}
	v := L.ToUserData(-1)
	if v == nil {
		L.Errorf("Invalid %s", key)
		return nil
	}
	L.Pop(1)

	return (*T)(v)
}

var task *ltask

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

	config.load(L, 1)

	if config.crashLog != nil {
		// TODO: set crash log
	}

	task.init(L, config)

	return 1
}

func ltaskInitTimer(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	if task.timer != nil {
		return L.Errorf("Timer can init only once")
	}
	task.timer = newTimer()

	return 0
}

func ltaskNewService(L *lua.State) int {
	task := getPtr[ltask](L, "LTASK_GLOBAL")
	label := L.CheckString(1)
	var sourceSz int
	source := L.CheckLString(2, &sourceSz)
	chunkName := L.CheckString(3)
	sid := L.OptInteger(4, 0)
	workerId := L.OptInteger(5, -1)

	id := task.services.newService(sid)

	if !task.newService(L, id, label, source, sourceSz, chunkName, workerId) {
		L.PushBoolean(false)
		L.Insert(-2)
		return 2
	}

	L.PushInteger(id)
	return 1
}

var bootInit atomic.Int32

func ltaskBootstrap(L *lua.State) int {
	if bootInit.Add(1) != 1 {
		return L.Errorf("ltask.bootstrap can only require once")
	}
	l := []luaLReg{
		{"init", ltaskInit},
		{"new_service", ltaskNewService},
		{"init_timer", ltaskInitTimer},
		// We don't need `init_socket` here, as it is proceed by Go runtime automatically.
	}

	luaLNewLib(L, l)
	return 1
}
