local ltask = require("ltask")

local saddr = ltask.spawn("webserver")

print("Spawn webserver at " .. saddr)

ltask.call(saddr, "start", {
  addr = "0.0.0.0",
  port = 9090,
  cgi = {
    user = "route.user",
  },
})

print("Webserver started at 0.0.0.0:9090")

local caddr = ltask.spawn("webclient")

print("Spawn webclient at " .. caddr)

print("Create user yuchanns")

local code, msg, header, body = ltask.call(caddr, "post", "http://127.0.0.1:9090/user", {
  name = "yuchanns",
  age = 32,
}, { ["Content-Type"] = "application/json" })
assert(code == 201)
print(body)

print("Get user yuchanns")
code, msg, header, body =
  ltask.call(caddr, "get", "http://127.0.0.1:9090/user?name=yuchanns", { ["Content-Type"] = "application/json" })
assert(code == 200)
print(body)

ltask.call(saddr, "quit")
ltask.call(caddr, "quit")

print("Webserver stopped")
