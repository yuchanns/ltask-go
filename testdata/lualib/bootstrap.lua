local boot = require("ltask.bootstrap")

local SERVICE_ROOT <const> = 1

local function bootstrap_root(initfunc, config)
	local sid = assert(boot.new_service("root", config.service_source, config.service_chunkname, SERVICE_ROOT))
	assert(sid == SERVICE_ROOT)
	boot.init_root(SERVICE_ROOT)

	local init_msg, sz = boot.pack("init", {
		initfunc = initfunc,
		name = "root",
		args = { config },
	})
end

local function start(config)
	boot.init(config.core)
	boot.init_timer()
	bootstrap_root(config.root_initfunc, config.root)
end

return {
	start = start,
}
