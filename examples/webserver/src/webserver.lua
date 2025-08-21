local ltask = require("ltask")

local http = Require("http")
local urllib = Require("http.url")

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

local fd

function S.start(conf)
  local addr = conf.addr or "127.0.0.1"
  local port = conf.port or 8080
  fd = assert(net.bind("tcp", addr, port))
  local cgi = {}
  if conf.cgi then
    for path, pname in pairs(conf.cgi) do
      local package, name = pname:match("(.-)|(.*)")
      if package then
        cgi[path] = { package = package, name = name }
      else
        cgi[path] = { package = pname }
      end
    end
  end
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
            local fullpath, query = urllib.parse(url)
            local root, path = fullpath:match("^/([^/]+)/?(.*)")
            local mod = cgi[root]
            if not mod then
              response(conn.write, 404, "Not Found")
            else
              local ok, m = xpcall(Require, debug.traceback, mod.package)
              if not ok then
                response(conn.write, 500, m)
              else
                if query then query = urllib.parse_query(query) end
                method = method:lower()
                m = m[mod.name] or m
                local f = m and m[method]
                if f == nil then
                  response(conn.write, 405, "Method Not Allowed")
                else
                  local data
                  ok, code, data, header = xpcall(f, debug.traceback, path, query, header, body)
                  if ok then
                    response(conn.write, code or 200, data, header)
                  else
                    response(conn.write, 500, code)
                  end
                end
              end
            end
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

function S.quit()
  if fd then
    net.close(fd)
    fd = nil
  end
  ltask.quit()
end

return S
