local ltask = require("ltask")

local S = setmetatable({}, { __gc = function() print("User exit") end })

print("User init :", ...)
local worker = ltask.worker_id()
print(string.format("User %d in worker %d", ltask.self(), worker))
ltask.worker_bind(worker) -- bind to current worker thread

local function coroutine_test()
  coroutine.yield("Coroutine yield")
  return "Coroutine end"
end

-- FIXME: Why the creation of coroutine raises root init_receipt error?
-- local co = coroutine.create(coroutine_test)
-- while true do
--   local ok, ret = coroutine.resume(co)
--   if ok then
--     print(ret)
--   else
--     break
--   end
-- end

function S.wait(ti)
  if ti < 10 then error("Error : " .. ti) end
  ltask.sleep(ti)
  return ti
end

function S.req(ti)
  print("Wait Req", ti)
  ltask.sleep(ti)
  print("wait resp", ti)
  return ti
end

function S.exit() ltask.quit() end

return S
