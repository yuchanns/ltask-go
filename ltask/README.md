# ltask

A lua task library that implements an n:m scheduler, runs M lua VMs on N OS threads.

## Caution

⚠️This library is **working in progress** 🚧 And APIs are not stable yet, maybe cause breaking changes many times. I make it public only for unlimited GitHub Actions minutes. It is not recommended to use at this moment.

## Instructions

### Usage

```bash
go get go.yuchanns.xyz/ltask
```

---
**ltask embedded Lua code**

ltask is composed of Go code and a set of Lua glue codes from `lualib` and `service`. The Lua glue code is embedded in the Go binary during compilation.

When using ltask, you can load the embedded Lua modules via `ltask.bootstrap` and `ltask`. For example, if you want to load the `bootstrap.lua` from `lualib`:

```lua
local boot = require("ltask.bootstrap")

local function searchpath(name)
  return assert(boot.searchpath(name, "lualib/?.lua"))
end
local bootstrap = boot.dofile(searchpath("bootstrap"))
-- now you can use the bootstrap to start ltask.
local ctx = bootstrap.start({})
bootstrap.wait(ctx)
```

For more information of usage about embedded Lua, check [`test/start.lua`](./test/start.lua).

---
**Go usage**

```go
func main() {
	lib, err := lua.New("/path/to/lua54.so")
	if err != nil {
		panic(err)
	}
	defer lib.Close()

	L, err := lib.NewState()
	if err != nil {
		panic(err)
	}
	defer L.Close()

	L.OpenLibs()

	// Open the ltask library
	ltask.OpenLibs(L, lib)

	// Now you can use ltask in Lua
	L.DoFile(`./main.lua`)

	// ...
}
```

**Lua usage**

```lua
-- user
local ltask = require "ltask"

local S = {}

print "User Start"

function S.ping(...)
	ltask.timeout(10, function() print(1) end)
	ltask.timeout(20, function() print(2) end)
	ltask.timeout(30, function() print(3) end)
	ltask.sleep(40) -- sleep 0.4 sec
	-- response
	return "PING", ...
end

return S

-- root
local function boot()
	print "Root Start"
	print(os.date("%c", (ltask.now())))
	local addr = S.spawn("user", "Hello")	-- spawn a new service `user`
	print(ltask.call(addr, "ping", "PONG"))	-- request "ping" message
end

boot()
```

## Credits

- [ltask](https://github.com/cloudwu/ltask) is a lua task library that implements an n:m scheduler, so that you can run M lua VMs on N OS threads.
