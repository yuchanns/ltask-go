local ltask = require("ltask")

local S = setmetatable({}, { __gc = function() print("User exit") end })

print("User init :", ...)

function S.exit() ltask.quit() end

return S
