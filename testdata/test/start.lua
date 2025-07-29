local function searchpath(name)
  return assert(package.searchpath(name, "testdata/lualib/?.lua"))
end

local function readall(path)
  local f <close> = assert(io.open(path))
  return f:read("a")
end

return function(config)
  local servicepath = searchpath("service")
  local root_config = {
    bootstrap = config.bootstrap,
    service_source = readall(servicepath),
    service_chunkname = "@" .. servicepath,
    initfunc = ([=[
local name = ...
package.path = [[${lua_path}]]
package.cpath = [[${lua_cpath}]]
local filename, err = package.searchpath(name, "${service_path}")
if not filename then
	return nil, err
end
return loadfile(filename)
]=]):gsub("%$%{([^}]*)%}", {
      lua_path = package.path,
      lua_cpath = package.cpath,
      service_path = config.service_path,
    }),
  }
  local bootstrap = dofile(searchpath("bootstrap"))
  local ctx = bootstrap.start({
    core = config.core or {},
    root = root_config,
    root_initfunc = root_config.initfunc,
    mainthread = config.mainthread,
  })
  print("ltask Start")
  bootstrap.wait(ctx)
end
