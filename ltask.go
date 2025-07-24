package ltask

import (
	"go.yuchanns.xyz/lua"
)

func OpenLibs(L *lua.State) {
	_ = L.GetGlobal("package")
	_, _ = L.GetField(-1, "preload")

	l := []luaLReg{
		{"ltask.bootstrap", ltaskBootstrap},
	}
	luaLSetFuncs(L, l)
	L.Pop(2)
}

type luaLReg struct {
	Name string
	Func lua.GoFunc
}

func luaLNewLib(L *lua.State, l []luaLReg) {
	L.NewTable()

	luaLSetFuncs(L, l)
}

func luaLSetFuncs(L *lua.State, l []luaLReg) {
	for _, i := range l {
		L.PushGoFunction(i.Func)
		L.SetField(-2, i.Name)
	}
}

type ltask struct {
	config *ltaskConfig
}
