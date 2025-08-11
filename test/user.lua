local ltask = require("ltask")

local S = setmetatable({}, { __gc = function() print("User exit") end })

print("User init :", ...)
local worker = ltask.worker_id()
print(string.format("User %d in worker %d", ltask.self(), worker))
ltask.worker_bind(worker) -- bind to current worker thread

function S.exit() ltask.quit() end

return S
