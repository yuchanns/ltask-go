local ltask = require("ltask")

local msg, sz = ltask.pack({
  name = "service",
  version = "1.0.0",
  description = "A simple service example",
  author = "Your Name",
  license = "MIT"
})

local data = ltask.unpack_remove(msg, sz)
assert(data.name == "service", "Service name mismatch")
assert(data.version == "1.0.0", "Service version mismatch")
assert(data.description == "A simple service example", "Service description mismatch")
assert(data.author == "Your Name", "Service author mismatch")
assert(data.license == "MIT", "Service license mismatch")

local function mainloop()
  print("Main loop running...")
end

mainloop()
