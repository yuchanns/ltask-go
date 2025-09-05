local boot = require("ltask.bootstrap")
local app = require("sapp")

local function searchpath(name) return assert(boot.searchpath(name, "lualib/?.lua")) end

function print(...)
  local t = table.pack(...)
  local str = {}
  for i = 1, t.n do
    str[#str + 1] = tostring(t[i])
  end
  local message = string.format("( ltask.bootstrap ) %s", table.concat(str, "\t"))
  boot.pushlog(boot.pack("info", message))
end

local M = {}

function M.start(config)
  local servicepath = searchpath("service")
  local root_config = {
    bootstrap = config.bootstrap,
    service_source = boot.readfile(servicepath),
    service_chunkname = "@" .. servicepath,
    initfunc = ([=[
package.path = [[${lua_path}]]
package.cpath = [[${lua_cpath}]]
local name = ...
local filename, err = package.searchpath(name, "${service_path}")
if filename then
  return loadfile(filename)
end
local ltask = require("ltask")
local filename, err = ltask.searchpath(name, "${service_path}")
if not filename then
  return nil, err
end
return ltask.loadfile(filename)
]=]):gsub("%$%{([^}]*)%}", {
      lua_path = package.path,
      lua_cpath = package.cpath,
      service_path = config.service_path,
    }),
  }
  local bootstrap = boot.dofile(searchpath("bootstrap"))
  local core = config.core or {}
  core.external_queue = core.external_queue or 4096
  local ctx = bootstrap.start({
    core = core,
    root = root_config,
    root_initfunc = ([=[
local ltask = require("ltask")
local name = ...
local filename, err = ltask.searchpath(name, "${service_path}")
if not filename then
  return nil, err
end
return ltask.loadfile(filename)
]=]):gsub("%$%{([^}]*)%}", {
      lua_path = package.path,
      lua_cpath = package.cpath,
      service_path = config.service_path,
    }),
    mainthread = config.mainthread,
  })
  print("ltask Start")

  local sender, sender_ud = boot.external_sender(ctx)
  local sendmessage = app.sendmessage
  local function send_message(...) sendmessage(sender, sender_ud, ...) end
  local unpackevent = app.unpackevent

  return {
    cleanup = function()
      send_message("cleanup")
      bootstrap.wait(ctx)
    end,
    frame = function(count) send_message("frame", count) end,
    event = function(...) send_message(unpackevent(...)) end,
  }
end

return M
