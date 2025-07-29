package ltask_test

import (
	"github.com/stretchr/testify/require"
	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

func (s *Suite) TestSerde(assert *require.Assertions, L *lua.State) {
	L.PushGoFunction(ltask.LuaSerdePack)
	L.SetGlobal("pack")
	L.PushGoFunction(ltask.LuaSerdeUnpack)
	L.SetGlobal("unpack")

	err := L.DoString(`
local t = {1, 2, 3, a = "hello", b = "world"}
local msg, sz = pack(t)
local t2 = unpack(msg, sz)
assert(#t2 == 3)
for i = 1, #t2 do
	assert(t2[i] == t[i])
end
	`)
	assert.NoError(err)
}
