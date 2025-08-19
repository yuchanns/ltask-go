local ltask = require("ltask")

local http = ImportPackage("http")

local S = {}

local network = ltask.queryservice("network")

function S.start(addr, port)
  addr = addr or "127.0.0.1"
  port = port or 8080
  local fd = assert(ltask.call(network, "bind", "tcp", addr, port))
  ltask.fork(function()
    while true do
      local s = assert(ltask.call(network, "listen", fd))
      ltask.fork(function()
        local data = ltask.call(network, "recv", s)
        print(data)
        ltask.call(network, "send", s, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello World!")
        ltask.call(network, "close", s)
      end)
    end
  end)
end

function S.quit() ltask.quit() end

return S
