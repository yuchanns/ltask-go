package ltask

import (
	"sync/atomic"

	"go.yuchanns.xyz/lua"
)

var lib = new(lua.Lib)

// OpenLibs opens ltask library.
// This is useful when you don't write Go code and just want to use ltask directly in Lua.
func OpenLibs(L *lua.State) {
	*lib = *L.Lib()

	L.GetGlobal("package")
	_ = L.GetField(-1, "preload")

	l := []*lua.Reg{
		{Name: "ltask.bootstrap", Func: lua.NewCallback(OpenBootstrap, L.Lib())},
	}
	L.SetFuncs(l, 0)
	L.Pop(2)
}

// OpenCore represents `require("ltask")` in Lua.
// This is useful when you want to bring ltask to your own lua-binding projects.
func OpenCore(L *lua.State) int {
	// ltask is expected to be required many times in many instances of lua.State,
	// so we should make sure all of its functions are global singletons.
	l := []*lua.Reg{
		{Name: "pack", Func: luaSerdePack},
		{Name: "unpack", Func: luaSerdeUnpack},
		{Name: "remove", Func: luaSerdeRemove},
		{Name: "unpack_remove", Func: luaSerdeUnpackRemove},
		{Name: "timer_sleep", Func: ltaskSleep},
		{Name: "loadfile", Func: ltaskLoadFile},
		{Name: "searchpath", Func: ltaskSearchPath},
		{Name: "readfile", Func: ltaskReadFile},
		{Name: "dofile", Func: ltaskDoFile},
	}

	L.NewLib(l)

	l2 := []*lua.Reg{
		{Name: "send_message", Func: lsendMessage},
		{Name: "recv_message", Func: lrecvMessage},
		{Name: "message_receipt", Func: lmessageReceipt},
		{Name: "self", Func: lself},
		{Name: "worker_id", Func: lworkerId},
		{Name: "worker_bind", Func: lworkerBind},
		{Name: "timer_add", Func: ltaskTimerAdd},
		{Name: "timer_update", Func: ltaskTimerUpdate},
		{Name: "now", Func: ltaskNow},
		{Name: "label", Func: ltaskLabel},
		{Name: "pushlog", Func: ltaskPushLog},
		{Name: "poplog", Func: ltaskPopLog},
		{Name: "eventinit", Func: ltaskEventInit},
	}

	if L.GetField(lua.LUA_REGISTRYINDEX, "LTASK_ID") != lua.LUA_TLIGHTUSERDATA {
		L.Errorf("No service id, the VM is not inited by ltask")
	}
	ud := L.ToUserData(-1)
	L.Pop(1)

	L.PushLightUserData(ud)
	L.SetFuncs(l2, 1)

	return 1
}

var bootInit atomic.Int32

// OpenBootstrap represents `require("ltask.bootstrap")` in Lua.
// It can only be required once across the whole program.
// This is useful when you want to bring ltask to your own lua-binding projects.
func OpenBootstrap(L *lua.State) int {
	if bootInit.Add(1) != 1 {
		return L.Errorf("ltask.bootstrap can only require once")
	}
	// ltask.bootstrap is expected to be required only once across the whole program,
	// so we don't need to make its functions global singletons.
	l := []*lua.Reg{
		{Name: "searchpath", Func: ltaskSearchPath},
		{Name: "readfile", Func: ltaskReadFile},
		{Name: "loadfile", Func: ltaskLoadFile},
		{Name: "dofile", Func: ltaskDoFile},
		{Name: "deinit", Func: lua.NewCallback(ltaskDeinit, L.Lib())},
		{Name: "run", Func: lua.NewCallback(ltaskRun, L.Lib())},
		{Name: "wait", Func: lua.NewCallback(ltaskWait, L.Lib())},
		{Name: "post_message", Func: lua.NewCallback(lpostMessage, L.Lib())},
		{Name: "new_service", Func: lua.NewCallback(ltaskNewService, L.Lib())},
		{Name: "init_timer", Func: lua.NewCallback(ltaskInitTimer, L.Lib())},
		{Name: "init_root", Func: lua.NewCallback(ltaskInitRoot, L.Lib())},
		{Name: "pushlog", Func: lua.NewCallback(ltaskBootPushLog, L.Lib())},
		// We don't need `init_socket` here, as it is proceed by Go runtime automatically.
		{Name: "pack", Func: luaSerdePack},
		{Name: "unpack", Func: luaSerdeUnpack},
		{Name: "remove", Func: luaSerdeRemove},
		{Name: "unpack_remove", Func: luaSerdeUnpackRemove},
		{Name: "external_sender", Func: lua.NewCallback(ltaskExternalSender, L.Lib())},
	}

	L.NewLib(l)

	L.PushLightUserData(L.ToUserData(L.UpValueIndex(1)))
	l2 := []*lua.Reg{
		{Name: "init", Func: lua.NewCallback(ltaskInit, L.Lib())},
	}
	L.SetFuncs(l2, 1)
	return 1
}

var rootInit atomic.Int32

// OpenRoot represents `require("ltask.root")` in Lua.
// It can only be required once across the whole program.
// This is useful when you want to bring ltask to your own lua-binding projects.
func OpenRoot(L *lua.State) int {
	if rootInit.Add(1) != 1 {
		return L.Errorf("ltask.root can only require once")
	}
	// ltask.root is expected to be required only once across the whole program,
	// so we don't need to make its functions global singletons.
	l := []*lua.Reg{
		{Name: "init_service", Func: lua.NewCallback(ltaskInitService, L.Lib())},
		{Name: "close_service", Func: lua.NewCallback(ltaskCloseService, L.Lib())},
	}

	L.NewLibTable(l)

	if L.GetField(lua.LUA_REGISTRYINDEX, "LTASK_ID") != lua.LUA_TLIGHTUSERDATA {
		return L.Errorf("No service id, the VM is not inited by ltask")
	}
	ud := (*serviceUd)(L.ToUserData(-1))
	L.Pop(1)

	L.PushLightUserData(ud)
	L.SetFuncs(l, 1)
	return 1
}
