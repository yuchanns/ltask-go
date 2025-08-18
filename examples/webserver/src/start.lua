local ltask = require("ltask")

local addr = ltask.spawn("webserver")

print("Spawn webserver at " .. addr)

ltask.call(addr, "start", "0.0.0.0", 9090)

print("Webserver started at 0.0.0.0:9090")

ltask.call(addr, "quit")

print("Webserver stopped")
