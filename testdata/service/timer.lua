local ltask = require("ltask")

local timer = {}

function timer.quit() ltask.quit() end

ltask.idle_handler(function()
  print("Idle handler called")
  ltask.timer_sleep(10)
end)

return timer
