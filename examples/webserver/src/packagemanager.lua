local registered = {}
local ltask = require("ltask")

local function sandbox_env(packagename)
  local env = {}
  local _LOADED = {}
  local _PRELOAD = package.preload
  local PATH <const> = "src/" .. packagename .. "/?.lua"

  function env.require(name)
    local _ = type(name) == "string"
      or error(("bad argument #1 to 'require' (string expected, got %s)"):format(type(name)))
    local p = _LOADED[name] or package.loaded[name]
    if p ~= nil then return p end
    do
      local func = _PRELOAD[name]
      if func then
        local r = func()
        if r == nil then r = true end
        package.loaded[name] = r
        return r
      end
    end
    local filename = name:gsub("%.", "/")
    local path = PATH:gsub("%?", filename)
    do
      local func, err = ltask.loadfile(path)
      if not func then error(("error loading module '%s' from file '%s':\n\t%s"):format(name, path, err)) end
      local r = func()
      if r == nil then r = true end
      _LOADED[name] = r
      return r
    end
    error(("module '%s' no found from package.preload['%s']"):format(name, name))
  end

  function env.loadfile(path)
    local filename = "/pkg/" .. packagename .. "/" .. path
    return ltask.loadfile(filename)
  end

  function env.dofile(path)
    local func, err = env.loadfile(path)
    if not func then error(err) end
    return func()
  end
  env.package = {
    loaded = _LOADED,
    preload = _PRELOAD,
  }
  return setmetatable(env, { __index = _G })
end

local function loadenv(name)
  local env = registered[name]
  if not env then
    env = sandbox_env(name)
    registered[name] = env
  end
  return env
end

function ImportPackage(name) return loadenv(name).require("main") end

return {
  loadenv = loadenv,
}
