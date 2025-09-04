local ltask = require("ltask")
local sapp = require("sapp")

local unpackmessage = sapp.unpackmessage

local S = {}

function S.quit() ltask.quit() end

local command = {}

command["cleanup"] = function() S.quit() end

function S.external(p)
  local what, arg1, arg2 = unpackmessage(p)
  print("external", what, arg1, arg2)
  if command[what] then command[what](arg1, arg2) end
end

ltask.fork(function() ltask.call(1, "external_forward", ltask.self(), "external") end)

-- for testing purpose, quit after 5 second
ltask.timeout(500, function() sapp.quit() end)

return S
