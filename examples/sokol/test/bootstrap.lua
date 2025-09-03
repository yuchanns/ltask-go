local ltask = require("ltask")
local sapp = require("sapp")

local unpackmessage = sapp.unpackmessage

local S = {}

function S.external(p)
  local what, arg1, arg2 = unpackmessage(p)
  print("external", what, arg1, arg2)
end

ltask.fork(function() ltask.call(1, "external_forward", ltask.self(), "external") end)

return S
