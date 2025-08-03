local SERVICE_ROOT <const> = 1

local MESSAGE_SYSTEM <const> = 0
local MESSAGE_REQUEST <const> = 1
local MESSAGE_RESPONSE <const> = 2
local MESSAGE_ERROR <const> = 3
local MESSAGE_SIGNAL <const> = 4
local MESSAGE_IDLE <const> = 5

local RECEIPT_DONE <const> = 1
local RECEIPT_ERROR <const> = 2
local RECEIPT_BLOCK <const> = 3

local SESSION_SEND_MESSAGE <const> = 0

local ltask = require("ltask")

local CURRENT_SERVICE <const> = ltask.self()
local CURRENT_SERVICE_LABEL <const> = ltask.label()

ltask.log = {}
for _, level in ipairs { "info", "error" } do
  ltask.log[level] = function(...)
    local t = table.pack(...)
    local str = {}
    for i = 1, t.n do
      str[#str + 1] = tostring(t[i])
    end
    local message = string.format("( %s ) %s", CURRENT_SERVICE_LABEL, table.concat(str, "\t"))
    ltask.pushlog(ltask.pack(level, message))
  end
end

ltask.log.info("startup" .. CURRENT_SERVICE)


local coroutine_create = coroutine.create
local coroutine_resume = coroutine.resume
local coroutine_close = coroutine.close
local coroutine_running = coroutine.running
local coroutine_yield = coroutine.yield

local yield_service = coroutine.yield
local yield_session = coroutine.yield

local function continue_session()
  coroutine_yield(true)
end

_G.coroutine = nil

local running_thread

local session_coroutine_suspend_lookup = {}
local session_coroutine_response = {}
local session_coroutine_address = {}

local SESSION = {
}

SESSION[MESSAGE_SYSTEM] = function(type, msg, sz)
  -- TODO: handle request
  print("Received system message:", type, msg, sz)
  local cmd, data = ltask.unpack(msg, sz)
  print("Command:", cmd)
  for k, v in pairs(data) do
    print("Data:", k, v)
  end
end

function ltask.post_message(addr, session, type, msg, sz)
  -- TODO: send message
end

local function post_response_message(addr, session, type, msg, sz)
  local receipt_type, receipt_msg, receipt_sz = ltask.post_message(addr, session, type, msg, sz)
  if receipt_type == RECEIPT_DONE then
    return
  end
  if receipt_type == RECEIPT_ERROR then
    ltask.remove(receipt_msg, receipt_sz)
  else
    -- RECEIPT_BLOCK
    -- TODO: send_block_message
  end
end

local function resume_session(co, ...)
  running_thread = co
  local ok, errobj = coroutine_resume(co, ...)
  running_thread = nil
  if ok then
    return errobj
  else
    local from = session_coroutine_address[co]
    local session = session_coroutine_response[co]

    -- term session
    session_coroutine_address[co] = nil
    session_coroutine_response[co] = nil

    -- traceback
    if from == nil or from == 0 or session == SESSION + SESSION_SEND_MESSAGE then
      ltask.log.error(tostring(errobj))
    else
      post_response_message(from, session, MESSAGE_ERROR, ltask.pack(errobj))
    end
    coroutine_close(co)
  end
end

local function wakeup_session(co, ...)
  local cont = resume_session(co, ...)
  while cont do
    yield_service()
    cont = resume_session(co)
  end
end

local coroutine_pool = setmetatable({}, { __mode = "kv" })

local function new_thread(f)
  local co = table.remove(coroutine_pool)
  if co == nil then
    co = coroutine_create(function(...)
      f(...)
      while true do
        f = nil
        coroutine_pool[#coroutine_pool + 1] = co
        f = coroutine_yield()
        f(coroutine_yield())
      end
    end)
  else
    coroutine_resume(co, f)
  end
  return co
end

local function new_session(f, from, session)
  local co = new_thread(f)
  session_coroutine_address[co] = from
  session_coroutine_response[co] = session
  return co
end

local quit = true -- set to true before the implementation is complete.

local function schedule_message()
  local from, session, type, msg, sz = ltask.recv_message()
  local f = SESSION[type]
  if f then
    -- new session for this message
    local co = new_session(f, from, session)
    wakeup_session(co, type, msg, sz)
  elseif from == nil then
    -- no message
    return
  else
    -- lookup suspend session
  end
end

local function mainloop()
  while true do
    schedule_message()
    if quit then
      ltask.log.info("quit.")
      return
    end
    yield_service()
  end
end

mainloop()
