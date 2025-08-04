local ltask = require("ltask")
local root = require("ltask.root")

local config = ...

local SERVICE_SYSTEM <const> = 0

local MESSAGE_ERROR <const> = 3

local MESSAGE_SCHEDULE_NEW <const> = 0
local MESSAGE_SCHEDULE_DEL <const> = 1

local RECEIPT_ERROR <const> = 2
local RECEIPT_BLOCK <const> = 3
local RECEIPT_RESPONCE <const> = 4

local S = {}

local anonymous_services = {}
local named_services = {}

local root_quit = ltask.quit
ltask.quit = function() end

do
  -- root init response to iteself
  local function init_receipt(type, msg, sz)
    local errobj = ltask.unpack_remove(msg, sz)
    if type == MESSAGE_ERROR then
      ltask.log.error("Root fatal:", table.concat(errobj, "\n"))
      -- writelog()
      root_quit()
    end
  end

  --- The session of root init message must be 1
  ltask.suspend(1, init_receipt)
end

local function register_service(address, name)
  if named_services[name] then
    error(("Name `%s` already exists."):format(name))
  end
  anonymous_services[address] = nil
  named_services[#named_services + 1] = name
  named_services[name] = address
  ltask.multi_wakeup("unique." .. name, address)
end

local function spawn(t)
  local type, address = ltask.post_message(SERVICE_SYSTEM, 0, MESSAGE_SCHEDULE_NEW)
  if type ~= RECEIPT_RESPONCE then
    error("send MESSAGE_SCHEDULE_NEW failed.")
  end
  anonymous_services[address] = true
  assert(root.init_service(address, t.name, config.service_source, config.service_chunkname, t.worker_id))
  ltask.syscall(address, "init", {
    initfunc = t.initfunc or config.initfunc,
    name = t.name,
    args = t.args or {},
  })
  return address
end

local unique = {}

local function spawn_unique(t)
  local address = named_services[t.name]
  if address then
    return address
  end
  local key = "unique." .. t.name
  if not unique[t.name] then
    unique[t.name] = true
    ltask.fork(function()
      local ok, addr = pcall(spawn, t)
      if not ok then
        local err = addr
        ltask.multi_interrupt(key, err)
        unique[t.name] = nil
        return
      end
      register_service(addr, t.name)
      unique[t.name] = nil
    end)
  end
end

local function quit()
  if next(anonymous_services) ~= nil then
    return
  end
  -- TODO: ltask.send
end

function S.spawn_service(t)
  if t.unique then
    return spawn_unique(t)
  end
  return spawn(t)
end

local function bootstrap()
  for _, t in ipairs(config.bootstrap) do
    S.spawn_service(t)
  end
end

ltask.dispatch(S)

bootstrap()

quit()
