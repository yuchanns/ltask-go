local start = require("test.start")

start({
  core = {
    -- debuglog = "=", -- stdout
  },
  service_path = "service/?.lua;test/?.lua",
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
    -- {
    -- 	name = "sockevent",
    -- 	unique = true,
    -- },
    {
      name = "bootstrap",
    },
  },
})
