# ltask-go

[![Behavior CI](https://github.com/yuchanns/ltask-go/actions/workflows/ltask_test.yml/badge.svg?branch=main)](https://github.com/yuchanns/ltask-go/actions)
[![Example CI](https://github.com/yuchanns/ltask-go/actions/workflows/ltask_example.yml/badge.svg?branch=main)](https://github.com/yuchanns/ltask-go/actions)
[![Go Reference](https://pkg.go.dev/badge/go.yuchanns.xyz/ltask.svg)](https://pkg.go.dev/go.yuchanns.xyz/ltask)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A Go implementation of [cloudwu/ltask](https://github.com/cloudwu/ltask) - a Lua task library with n:m scheduler that enables running M Lua VMs on N OS threads.

## ⚠️ Caution

This library is currently **working in progress** 🚧. APIs are not stable and may undergo breaking changes. It is not recommended for production use at this time.

## Features

- **N:M Scheduler**: Run multiple Lua VMs efficiently across OS threads
- **Bootstrap Architecture**: Proper service initialization through bootstrap process
- **Embedded Lua Runtime**: Built-in Lua support with embedded glue code
- **Service Management**: Lightweight service model for concurrent tasks
- **Cross-Platform**: Works on Windows, Linux, and macOS

## Quick Start

### Installation

```bash
go get go.yuchanns.xyz/ltask
```

### Basic Usage Pattern

ltask-go follows the bootstrap pattern used in the test files. Create a main Lua file that requires the bootstrap module and starts the system:

```lua
-- main.lua
local start = require("test.start")

start({
    core = {
        worker = 3,  -- Number of worker threads
        -- debuglog = "=",  -- Uncomment for debug output
    },
    service_path = "service/?.lua",  -- Service search path
    bootstrap = {
        {
            name = "timer",
            unique = true,
            builtin = true,
        },
        {
            name = "logger", 
            unique = true,
            builtin = true,
        },
        {
            name = "my_service",  -- Your custom service
            unique = false,
        }
    }
})
```

### Creating a Service

Create a service file. The service path is configurable in the bootstrap setup:

```lua
-- service/my_service.lua
local ltask = require "ltask"

local my_service = {}

function my_service.echo(message)
    print("Service received:", message)
    return "Echo: " .. message
end

function my_service.ping()
    return "pong"
end

return my_service
```

The service will be automatically loaded when referenced in the bootstrap configuration.

## Core Concepts

### Bootstrap Process

ltask-go requires a proper bootstrap sequence:

1. **Require bootstrap**: `local boot = require("ltask.bootstrap")`
2. **Search embedded files**: Use `boot.searchpath()` to find embedded Lua modules
3. **Load bootstrap**: `boot.dofile()` to load the main bootstrap module
4. **Start system**: `bootstrap.start()` with configuration
5. **Wait for completion**: `bootstrap.wait()` to keep the system running

### Service Configuration

Services are configured through the bootstrap process. The loading behavior is completely customizable via `initfunc`:

- **unique**: Whether only one instance is allowed  
- **service_path**: Where to search for service files
- **initfunc**: Custom function to control how services are loaded (completely flexible)

Different examples show different `initfunc` implementations:
- **Root test**: Uses `ltask.searchpath` and `ltask.loadfile` for embedded services
- **Sokol example**: Uses standard `package.searchpath` and `loadfile` for external services

## Examples

Explore the provided examples for proper usage patterns:

- **[`test.lua`](./test.lua)** - Main test entry point
- **[`test/start.lua`](./test/start.lua)** - Bootstrap implementation
- **[`test/user.lua`](./test/user.lua)** - Example service implementation
- **[`examples/`](./examples/)** - Application examples, such as [sokol](https://github.com/floooh/sokol)-integration

## Running the Tests

```bash
# Clone the repository and submodules
git clone --recurse-submodules https://github.com/yuchanns/ltask-go
git clone --recurse-submodules https://github.com/yuchanns/lua

go work init
go work use ./ltask-go
go work use ./lua

# Build Lua library
cd lua && luamake && cd -

# Run tests
cd ltask-go && go test -v ./...
```

## Credits

This project is a Go rewrite of [cloudwu/ltask](https://github.com/cloudwu/ltask), originally created by [云风](https://github.com/cloudwu).

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
