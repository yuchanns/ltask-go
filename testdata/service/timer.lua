local ltask = require("ltask")

local timer = {}

function timer.quit()
	ltask.quit()
end

print("service timer started...")

return timer
