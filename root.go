package ltask

import (
	"sync/atomic"

	"go.yuchanns.xyz/lua"
)

var rootInit atomic.Int32

func ltaskRootOpen(L *lua.State) int {
	if rootInit.Add(1) != 1 {
		return L.Errorf("ltask.root can only require once")
	}
	l := []*lua.Reg{}

	L.NewLib(l)
	return 1
}
