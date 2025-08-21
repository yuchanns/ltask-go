local internal = Require("http.internal")

local string = string
local type = type

local httpd = {}

local http_status_msg = {
  [100] = "Continue",
  [101] = "Switching Protocols",
  [200] = "OK",
  [201] = "Created",
  [202] = "Accepted",
  [203] = "Non-Authoritative Information",
  [204] = "No Content",
  [205] = "Reset Content",
  [206] = "Partial Content",
  [300] = "Multiple Choices",
  [301] = "Moved Permanently",
  [302] = "Found",
  [303] = "See Other",
  [304] = "Not Modified",
  [305] = "Use Proxy",
  [307] = "Temporary Redirect",
  [400] = "Bad Request",
  [401] = "Unauthorized",
  [402] = "Payment Required",
  [403] = "Forbidden",
  [404] = "Not Found",
  [405] = "Method Not Allowed",
  [406] = "Not Acceptable",
  [407] = "Proxy Authentication Required",
  [408] = "Request Time-out",
  [409] = "Conflict",
  [410] = "Gone",
  [411] = "Length Required",
  [412] = "Precondition Failed",
  [413] = "Request Entity Too Large",
  [414] = "Request-URI Too Large",
  [415] = "Unsupported Media Type",
  [416] = "Requested range not satisfiable",
  [417] = "Expectation Failed",
  [500] = "Internal Server Error",
  [501] = "Not Implemented",
  [502] = "Bad Gateway",
  [503] = "Service Unavailable",
  [504] = "Gateway Time-out",
  [505] = "HTTP Version not supported",
}

local function readall(readbytes, bodylimit)
  local tmpline = {}
  local body = internal.recvheader(readbytes, tmpline, "")
  if not body then
    return 413 -- Request Entity Too Large
  end
  local request = assert(tmpline[1])
  local method, url, httpver = request:match("^(%a+)%s+(.-)%s+HTTP/([%d%.]+)$")
  assert(method and url and httpver)
  httpver = assert(tonumber(httpver))
  if httpver < 1.0 or httpver > 1.1 then
    return 505 -- HTTP Version not supported
  end
  local header = internal.parseheader(tmpline, 2, {})
  if not header then
    return 400 -- Bad request
  end
  local length = header["content-length"]
  if length then length = tonumber(length) end
  local mode = header["transfer-encoding"]
  if mode then
    if mode ~= "identity" and mode ~= "chunked" then
      return 501 -- Not Implemented
    end
  end

  if mode == "chunked" then
    body, header = internal.recvchunkedbody(readbytes, bodylimit, header, body)
    if not body then return 413 end
  else
    -- identity mode
    if length then
      if bodylimit and length > bodylimit then return 413 end
      if #body >= length then
        body = body:sub(1, length)
      else
        local padding = readbytes(length - #body)
        body = body .. padding
      end
    end
  end

  return 200, url, method, header, body
end

function httpd.read_request(...)
  local ok, code, url, method, header, body = pcall(readall, ...)
  if ok then
    return code, url, method, header, body
  else
    return nil, code
  end
end

local function writeall(writefunc, statuscode, bodyfunc, header)
  local statusline = string.format("HTTP/1.1 %03d %s\r\n", statuscode, http_status_msg[statuscode] or "")
  writefunc(statusline)
  if header then
    for k, v in pairs(header) do
      if type(v) == "table" then
        for _, v in ipairs(v) do
          writefunc(string.format("%s: %s\r\n", k, v))
        end
      else
        writefunc(string.format("%s: %s\r\n", k, v))
      end
    end
  end
  local t = type(bodyfunc)
  if t == "string" then
    writefunc(string.format("content-length: %d\r\n\r\n", #bodyfunc))
    writefunc(bodyfunc)
  elseif t == "function" then
    writefunc("transfer-encoding: chunked\r\n")
    while true do
      local s = bodyfunc()
      if s then
        if s ~= "" then
          writefunc(string.format("\r\n%x\r\n", #s))
          writefunc(s)
        end
      else
        writefunc("\r\n0\r\n\r\n")
        break
      end
    end
  else
    assert(t == "nil")
    writefunc("\r\n")
  end
end

function httpd.write_response(...) return xpcall(writeall, debug.traceback, ...) end

local function build_request(method, uri, header, body)
  local t = {}
  table.insert(t, string.format("%s %s HTTP/1.1\r\n", method, uri:match("^https?://[^/]+(/.*)$") or uri))
  if header then
    for k, v in pairs(header) do
      if type(v) == "table" then
        for _, vv in ipairs(v) do
          table.insert(t, string.format("%s: %s\r\n", k, vv))
        end
      else
        table.insert(t, string.format("%s: %s\r\n", k, v))
      end
    end
  end
  if not header or not header["Host"] then
    local host = uri:match("^https?://([^:/]+)") or ""
    table.insert(t, string.format("Host: %s\r\n", host))
  end
  if type(body) == "string" then
    table.insert(t, string.format("Content-Length: %d\r\n", #body))
    table.insert(t, "\r\n")
    table.insert(t, body)
  elseif type(body) == "function" then
    table.insert(t, "Transfer-Encoding: chunked\r\n\r\n")
    while true do
      local chunk = body()
      if not chunk or chunk == "" then
        table.insert(t, "0\r\n\r\n")
        break
      else
        table.insert(t, string.format("%x\r\n%s\r\n", #chunk, chunk))
      end
    end
  else
    table.insert(t, "\r\n")
  end
  return table.concat(t)
end

local function read_response(readbytes, bodylimit)
  local tmpline = {}
  local body = internal.recvheader(readbytes, tmpline, "")
  if not body then
    return 413 -- Request Entity Too Large
  end
  local status = assert(tmpline[1])
  local httpver, statuscode, statusmsg = status:match("^HTTP/([%d%.]+)%s+(%d+)%s*(.*)$")
  assert(httpver and statuscode)
  statuscode = assert(tonumber(statuscode))
  httpver = assert(tonumber(httpver))
  if httpver < 1.0 or httpver > 1.1 then
    return 505 -- HTTP Version not supported
  end
  local header = internal.parseheader(tmpline, 2, {})
  if not header then
    return 400 -- Bad response
  end
  local length = header["content-length"]
  if length then length = tonumber(length) end
  local mode = header["transfer-encoding"]
  if mode then
    if mode ~= "identity" and mode ~= "chunked" then
      return 501 -- Not Implemented
    end
  end

  if mode == "chunked" then
    body, header = internal.recvchunkedbody(readbytes, bodylimit, header, body)
    if not body then return 413 end
  else
    -- identity mode
    if length then
      if bodylimit and length > bodylimit then return 413 end
      if #body >= length then
        body = body:sub(1, length)
      else
        local padding = readbytes(length - #body)
        body = body .. padding
      end
    end
  end

  return statuscode, statusmsg, header, body
end

function httpd.read_response(...)
  local ok, code, msg, header, body = pcall(read_response, ...)
  if ok then
    return code, msg, header, body
  else
    return nil, code
  end
end

function httpd.request(writebytes, readbytes, method, url, header, body, bodylimit)
  method = method:upper()
  local uri = url

  local reqdata = build_request(method, uri, header, body)
  writebytes(reqdata)
  local code, msg, resp_header, resp_body = httpd.read_response(readbytes, bodylimit)
  return code, msg, resp_header, resp_body
end

return httpd
