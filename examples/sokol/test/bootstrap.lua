local ltask = require("ltask")
local sapp = require("sapp")

local unpackmessage = sapp.unpackmessage

local S = {}

function S.quit() ltask.quit() end

local command = {}

command["cleanup"] = function()
  print("cleanup")
  S.quit()
end
command["frame"] = function(count) print("frame", count) end
command["event"] = function(name, data) print("event", name, data) end

function S.external(p)
  local what, arg1, arg2 = unpackmessage(p)
  if command[what] then command[what](arg1, arg2) end
end

ltask.fork(function() ltask.call(1, "external_forward", ltask.self(), "external") end)

-- for testing purpose, quit after 5 second
ltask.timeout(500, function() sapp.quit() end)

return S
