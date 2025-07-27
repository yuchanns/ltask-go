# ltask-go

A showcase of how to build a lua library with [go.yuchanns.xyz/lua](https://github.com/yuchanns/lua) the lua go-binding.

## Caution

⚠️This library is **working in progress** 🚧 And APIs are not stable yet, maybe cause breaking changes many times. I make it public only for unlimited GitHub Actions minutes. It is not recommended to use at this moment.

## Instructions

### Usage

```bash
go get go.yuchanns.xyz/ltask
```

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

### Devlopment

We use Go Workspace to manage in early development stage, so you can use the following commands to run the tests.

1. **Clone the repository**
```bash
mkdir ltask_workspace

cd ltask_workspace

git clone https://github.com/yuchanns/ltask-go

git clone --recurse-submodules https://github.com/yuchanns/lua
```

2. **Prepare the dynamic artifacts**
```bash
cd lua && make lua54 && cd -
```

3. **Run the tests**
```bash
cd ltask-go && go test -v ./...
```

## Credits

- [ltask](https://github.com/cloudwu/ltask) is a lua task library that implements an n:m scheduler, so that you can run M lua VMs on N OS threads.
