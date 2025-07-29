---@class ltask.bootstrap
---@field init fun(config: table)
---@field init_timer fun()
---@field init_root fun(sid: integer)
---@field new_service fun(name: string, source: string, chunkname: string, sid: integer): integer
---@field pack fun(name: string, args: table): string, integer
---@field post_message fun(msg: ltask.message)
---@field run fun(mainthread: integer|nil): userdata
---@field wait fun(ctx: userdata)
---@field deinit fun()
local boot = require("ltask.bootstrap")

---@class ltask.message
---@field from integer
---@field to integer
---@field session integer
---@field type integer
---@field message string
---@field size integer

local SERVICE_ROOT <const> = 1
local MESSSAGE_SYSTEM <const> = 0

---@param initfunc string
---@param config bootconfig
local function bootstrap_root(initfunc, config)
  local sid = assert(boot.new_service("root", config.service_source, config.service_chunkname, SERVICE_ROOT))
  assert(sid == SERVICE_ROOT)
  boot.init_root(SERVICE_ROOT)

  local init_msg, sz = boot.pack("init", {
    initfunc = initfunc,
    name = "root",
    args = { config },
  })
  boot.post_message({
    from = SERVICE_ROOT,
    to = SERVICE_ROOT,
    session = 1, -- 1 for root init
    type = MESSSAGE_SYSTEM,
    message = init_msg,
    size = sz,
  })
end

---@class bootconfig
---@field core table
---@field root_initfunc string
---@field root table
---@field service_source string
---@field service_chunkname string
---@field mainthread integer|nil

---@param config bootconfig
local function start(config)
  boot.init(config.core)
  boot.init_timer()
  bootstrap_root(config.root_initfunc, config.root)
  return boot.run(config.mainthread)
end

---@param ctx userdata
local function wait(ctx)
  boot.wait(ctx)
  boot.deinit()
end

return {
  start = start,
  wait = wait,
}
