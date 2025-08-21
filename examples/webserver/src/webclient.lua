local ltask = require("ltask")
local json = require("json")

local http = Require("http")
local urllib = Require("http.url")

local network = ltask.queryservice("network")

local net = {
  connect = function(...) return ltask.call(network, "connect", ...) end,
  send = function(...) return ltask.call(network, "send", ...) end,
  recv = function(...) return ltask.call(network, "recv", ...) end,
  close = function(...) return ltask.call(network, "close", ...) end,
}

local S = {}

local function parse_fullpath(fullpath)
  local scheme, rest = fullpath:match("^(%a[%w]*)://(.+)$")
  if not scheme then
    scheme = "http"
    rest = fullpath
  end

  local host, port, path
  host, port, path = rest:match("^(.-):(%d+)(/.*)$") -- host:port/path
  if not host then
    host, path = rest:match("^(.-)(/.*)$") -- host/path
  end
  if not host then
    host, port = rest:match("^(.-):(%d+)$") -- host:port
  end
  if not host then host = rest end

  if not port then port = "80" end
  if not path then path = "/" end

  return scheme, host, tonumber(port), path
end

local socket_error = setmetatable({}, { __tostring = function() return "[Socket Error]" end })

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

function S.get(url, header)
  local fullpath, _ = urllib.parse(url)
  local scheme, host, port, _ = parse_fullpath(fullpath)
  assert(scheme == "http" or scheme == "https", "Unsupported scheme: " .. scheme)
  if not host then return nil, "Invalid URL: " .. url end
  local fd, err = net.connect("tcp", host, port)
  if not fd then return nil, "Connect to " .. host .. ":" .. port .. " failed: " .. err end
  local conn = new_conn(fd)
  return http.request(conn.write, conn.recv, "GET", url, header)
end

function S.post(url, data, header)
  local fullpath, _ = urllib.parse(url)
  local scheme, host, port, _ = parse_fullpath(fullpath)
  assert(scheme == "http" or scheme == "https", "Unsupported scheme: " .. scheme)
  if not host then return nil, "Invalid URL: " .. url end
  local fd, err = net.connect("tcp", host, port)
  if not fd then return nil, "Connect to " .. host .. ":" .. port .. " failed: " .. err end
  local conn = new_conn(fd)
  data = json.encode(data)
  return http.request(conn.write, conn.recv, "POST", url, header, data)
end

function S.quit() ltask.quit() end

return S
