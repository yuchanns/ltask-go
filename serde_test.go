package ltask_test

import (
	"github.com/smasher164/mem"
	"github.com/stretchr/testify/require"
	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

func (s *Suite) TestSerde(assert *require.Assertions, L *lua.State) {
	L.PushCFunction(lua.NewCallback(ltask.LuaSerdePack))
	L.SetGlobal("pack")
	L.PushCFunction(lua.NewCallback(ltask.LuaSerdeUnpack))
	L.SetGlobal("unpack")
	L.PushCFunction(lua.NewCallback(ltask.LuaSerdeUnpackRemove))
	L.SetGlobal("unpack_remove")
	L.PushCFunction(lua.NewCallback(ltask.LuaSerdeRemove))
	L.SetGlobal("remove")

	testFunc1 := func(L *lua.State) int {
		L.PushString("test function 1")
		return 1
	}
	testFunc2 := func(L *lua.State) int {
		L.PushInteger(42)
		return 1
	}

	L.PushCFunction(lua.NewCallback(testFunc1))
	L.SetGlobal("test_func1")
	L.PushCFunction(lua.NewCallback(testFunc2))
	L.SetGlobal("test_func2")

	err := L.DoString(`
local t = {1, 2, 3, a = "hello", b = "world"}
local msg, sz = pack(t)
local t2 = unpack(msg, sz)
assert(#t2 == 3)
for i = 1, #t2 do
	assert(t2[i] == t[i])
end
assert(t2.a == "hello")
assert(t2.b == "world")
local t3 = unpack_remove(msg, sz)
assert(#t3 == 3)
for i = 1, #t3 do
	assert(t3[i] == t[i])
end
assert(t3.a == "hello")
assert(t3.b == "world")
	`)
	assert.NoError(err)

	err = L.DoString(`
local msg, sz = pack(test_func1)
local result = unpack(msg, sz)
assert(type(result) == "function")
assert(result() == "test function 1")

local msg2, sz2 = pack(test_func2)
local result2 = unpack(msg2, sz2)
assert(type(result2) == "function")
assert(result2() == 42)
	`)
	assert.NoError(err)

	err = L.DoString(`

		local t = {
			name = "function table",
			func1 = test_func1,
			func2 = test_func2,
			data = {1, 2, 3}
		}

	local msg, sz = pack(t)
	local result = unpack(msg, sz)
	assert(result.name == "function table")
	assert(type(result.func1) == "function")
	assert(type(result.func2) == "function")
	assert(result.func1() == "test function 1")
	assert(result.func2() == 42)
	assert(#result.data == 3)

		`)
	assert.NoError(err)

	// In general, light userdata should be alived for the duration of the Lua state.
	// If we use Go pointers here, it cannot pass the checkptr
	// So we use mem.Alloc to allocate memory for the light userdata.
	testPtr1 := mem.Alloc(1024 * 1024)
	testPtr2 := mem.Alloc(1024)

	defer mem.Free(testPtr1)
	defer mem.Free(testPtr2)

	L.PushLightUserData(testPtr1)
	L.SetGlobal("test_ptr1")
	L.PushLightUserData(testPtr2)
	L.SetGlobal("test_ptr2")

	err = L.DoString(`

	local msg, sz = pack(test_ptr1)
	local result = unpack(msg, sz)
	assert(type(result) == "userdata")
	assert(result == test_ptr1)

	local msg2, sz2 = pack(test_ptr2)
	local result2 = unpack(msg2, sz)
	assert(type(result2) == "userdata")
	assert(result2 == test_ptr2)

		`)
	assert.NoError(err)

	err = L.DoString(`

		local t = {
			name = "userdata table",
			ptr1 = test_ptr1,
			ptr2 = test_ptr2,
			count = 2
		}

	local msg, sz = pack(t)
	local result = unpack(msg, sz)
	assert(result.name == "userdata table")
	assert(type(result.ptr1) == "userdata")
	assert(type(result.ptr2) == "userdata")
	assert(result.ptr1 == test_ptr1)
	assert(result.ptr2 == test_ptr2)
	assert(result.count == 2)

		`)
	assert.NoError(err)

	err = L.DoString(`

		local complex = {
			functions = {
				f1 = test_func1,
				f2 = test_func2
			},
			pointers = {
				p1 = test_ptr1,
				p2 = test_ptr2
			},
			data = {
				numbers = {1, 2, 3.14, -100},
				strings = {"hello", "world", ""},
				booleans = {true, false},
				nested = {
					level1 = {
						level2 = {
							func = test_func1,
							ptr = test_ptr1,
							value = "deep"
						}
					}
				}
			}
		}

	local msg, sz = pack(complex)
	local result = unpack(msg, sz)

	assert(type(result.functions.f1) == "function")
	assert(type(result.functions.f2) == "function")
	assert(result.functions.f1() == "test function 1")
	assert(result.functions.f2() == 42)

	assert(type(result.pointers.p1) == "userdata")
	assert(type(result.pointers.p2) == "userdata")
	assert(result.pointers.p1 == test_ptr1)
	assert(result.pointers.p2 == test_ptr2)

	assert(#result.data.numbers == 4)
	assert(result.data.numbers[1] == 1)
	assert(math.abs(result.data.numbers[3] - 3.14) < 0.00001)
	assert(result.data.strings[1] == "hello")
	assert(result.data.booleans[1] == true)
	assert(result.data.booleans[2] == false)

	assert(type(result.data.nested.level1.level2.func) == "function")
	assert(result.data.nested.level1.level2.func() == "test function 1")
	assert(result.data.nested.level1.level2.ptr == test_ptr1)
	assert(result.data.nested.level1.level2.value == "deep")

		`)
	assert.NoError(err)

	err = L.DoString(`

	local shared_func = test_func1
	local shared_ptr = test_ptr1

		local t = {
			a = {func = shared_func, ptr = shared_ptr},
			b = {func = shared_func, ptr = shared_ptr}
		}

	local msg, sz = pack(t)
	local result = unpack(msg, sz)

	assert(result.a.func == result.b.func)
	assert(result.a.func() == "test function 1")
	assert(result.b.func() == "test function 1")

	assert(result.a.ptr == result.b.ptr)
	assert(result.a.ptr == test_ptr1)
	assert(result.b.ptr == test_ptr1)

		`)
	assert.NoError(err)

	err = L.DoString(`
	local msg, sz = pack(test_func1, test_ptr1, "string", 123, true, test_func2, test_ptr2)
	local f1, p1, str, num, bool, f2, p2 = unpack(msg, sz)

	assert(type(f1) == "function")
	assert(f1() == "test function 1")
	assert(type(p1) == "userdata")
	assert(p1 == test_ptr1)
	assert(str == "string")
	assert(num == 123)
	assert(bool == true)
	assert(type(f2) == "function")
	assert(f2() == 42)
	assert(type(p2) == "userdata")
	assert(p2 == test_ptr2)
		`)
	assert.NoError(err)

	err = L.DoString(`
	local t = {
		name = "circular",
		func = test_func1,
		ptr = test_ptr1
	}

	local msg, sz = pack(t)
	local result = unpack(msg, sz)

	assert(result.name == "circular")
	assert(type(result.func) == "function")
	assert(result.func() == "test function 1")
	assert(result.ptr == test_ptr1)
		`)
	assert.NoError(err)
}

func (s *Suite) TestSerdeFunction(assert *require.Assertions, L *lua.State) {
	L.PushCFunction(lua.NewCallback(ltask.LuaSerdePack))
	L.SetGlobal("pack")
	L.PushCFunction(lua.NewCallback(ltask.LuaSerdeUnpack))
	L.SetGlobal("unpack")

	err := L.DoString(`
return function()
  local upvalue = "test"
	return function()
	  return upvalue
	end
end
	`)
	assert.NoError(err)

	L.SetGlobal("closure_func")

	err = L.DoString(`
pack(closure_func)
	`)
	assert.Error(err)

	err = L.DoString(`
function lua_func()
  return "lua function"
end
pack(lua_func)
	`)
	assert.Error(err)
}
