local ltask = require("ltask")

local http = Require("http")

local S = {}

local network = ltask.queryservice("network")

local net = {
  bind = function(...) return ltask.call(network, "bind", ...) end,
  listen = function(...) return ltask.call(network, "listen", ...) end,
  send = function(...) return ltask.call(network, "send", ...) end,
  recv = function(...) return ltask.call(network, "recv", ...) end,
  close = function(...) return ltask.call(network, "close", ...) end,
}

local socket_error = setmetatable({}, { __tostring = function() return "[Socket Error]" end })

local function response(write, ...)
  local ok, err = http.write_response(write, ...)
  if not ok then
    if err ~= socket_error then print(string.format("%s", err)) end
  end
end

local function new_conn(fd)
  local conn = {}
  function conn.write(data)
    local ok, err = net.send(fd, data)
    if not ok then
      if err ~= socket_error then print(string.format("Send error: %s", err)) end
      return false, err
    end
    return true
  end

  function conn.recv()
    local data, err = net.recv(fd)
    if not data then
      if err ~= socket_error then print(string.format("Receive error: %s", err)) end
      return nil, err
    end
    return data
  end

  function conn.close() net.close(fd) end

  return conn
end

function S.start(addr, port)
  addr = addr or "127.0.0.1"
  port = port or 8080
  local fd = assert(net.bind("tcp", addr, port))
  ltask.fork(function()
    while true do
      local s = assert(net.listen(fd))
      local conn = new_conn(s)
      ltask.fork(function()
        local code, url, method, header, body = http.read_request(conn.recv)
        if code then
          if code ~= 200 then
            response(conn.write, code)
          else
            print("Received request:", method, url)
            response(conn.write, 200, "Hello World!")
          end
        else
          if url == socket_error then
            print("socket closed")
          else
            print(url)
          end
        end
        conn.close()
      end)
    end
  end)
end

function S.quit() ltask.quit() end

return S
