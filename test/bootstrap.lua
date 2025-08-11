local ltask = require("ltask")

local arg = ...

print("Bootstrap Begin")
print(os.date("%c", (ltask.now())))
local addr = ltask.spawn("user", "Hello")

print("Spawn user", addr)

--- test request

local tasks = {
  { ltask.call, addr, "wait", 30, id = 1 },
  { ltask.call, addr, "wait", 20, id = 2 },
  { ltask.call, addr, "wait", 10, id = 3 },
  { ltask.call, addr, "wait", 5, id = 4 },
  { ltask.sleep, 25 },
}

for req, resp in ltask.parallel(tasks) do
  if not req.id then break end
  if not resp.error then
    print("REQ", req.id, resp[1])
  else
    print("ERR", req.id, resp.error)
  end
end

ltask.send(addr, "exit")

print("Bootstrap End")
