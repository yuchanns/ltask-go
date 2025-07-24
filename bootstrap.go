package ltask

import (
	"fmt"

	"go.yuchanns.xyz/lua"
)

func ltaskInit(L *lua.State) int {
	fmt.Println("bootstrap initialized")
	return 0
}

func ltaskBootstrap(L *lua.State) int {
	l := []luaLReg{
		{"init", ltaskInit},
	}

	luaLNewLib(L, l)
	return 1
}
