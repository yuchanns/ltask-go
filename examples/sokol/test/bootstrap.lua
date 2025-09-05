local ltask = require("ltask")
local sapp = require("sapp")

local unpackmessage = sapp.unpackmessage

local S = {}

function S.quit() ltask.quit() end

local command = {}

function command.cleanup()
  print("cleanup")
  S.quit()
end

function command.frame(count) print("frame", count) end

function command.mouse_move(x, y) print("mouse_move", x, y) end

function command.mouse_button(button, action) print("mouse_button", button, action) end

function command.mouse_scroll(x, y) print("mouse_scroll", x, y) end

function command.mouse(type) print("mouse", type) end

function command.window_resize(w, h) print("window_resize", w, h) end

function command.window(type) print("window", type) end

function command.char(c) print("char", c) end

function command.key(type) print("key", type) end

function command.message(type) print("message", type) end

function S.external(p)
  local what, arg1, arg2 = unpackmessage(p)
  if command[what] then command[what](arg1, arg2) end
end

ltask.fork(function() ltask.call(1, "external_forward", ltask.self(), "external") end)

-- for testing purpose, quit after 5 second
ltask.timeout(500, function() sapp.quit() end)

return S
