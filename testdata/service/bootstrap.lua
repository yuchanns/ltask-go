local ltask = require("ltask")

local arg = ...

print("Bootstrap Begin")
print(os.date("%c", (ltask.now())))
local addr = ltask.spawn("user", "Hello")

print("Spawn user", addr)

ltask.send(addr, "exit")

print("Bootstrap End")
