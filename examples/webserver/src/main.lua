local boot = require("ltask.bootstrap")

local func = boot.loadfile(assert(boot.searchpath("test", "src/?.lua")))
func()
