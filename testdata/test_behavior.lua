local start = require("testdata.test.start")

start({
  core = {
    debuglog = "=", -- stdout
  },
  service_path = "testdata/service/?.lua;test/?.lua",
  bootstrap = {
    {
      name = "timer",
      unique = true,
    },
    -- {
    -- 	name = "logger",
    -- 	unique = true,
    -- },
    -- {
    -- 	name = "sockevent",
    -- 	unique = true,
    -- },
    -- {
    -- 	name = "bootstrap",
    -- },
  },
})
