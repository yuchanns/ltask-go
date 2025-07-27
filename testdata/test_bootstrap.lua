local boot = require("ltask.bootstrap")

local function searchpath(name)
  return assert(package.searchpath(name, "testdata/lualib/?.lua"))
end

local function readall(path)
  local f <close> = assert(io.open(path))
  return f:read "a"
end

boot.init({})
boot.init_timer()

local SERVICE_ROOT <const> = 1

local servicepath = searchpath "service"
local service_source = readall(servicepath)
local service_chunkname = "@" .. servicepath

local sid = assert(boot.new_service("root", service_source, service_chunkname, SERVICE_ROOT))
assert(sid == SERVICE_ROOT)
