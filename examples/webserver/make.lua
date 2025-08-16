local lm = require("luamake")

lm.base_dir = lm:path(".")

local bindir = "build/bin"
lm.bindir = bindir

lm:conf({
  compile_commands = "build",
  mode = "debug",
  c = "c17",
  cxx = "c++20",
  visibility = "default",
  windows = {
    defines = {
      "_CRT_SECURE_NO_WARNINGS",
      "_WIN32_WINNT=0x0602",
    },
    flags = {
      "/utf-8",
      "/arch:AVX2",
    },
  },
})

lm:import("clibs/lua/make.lua")
lm.EXE = "lua"
lm:import("bee/make.lua")

if lm.os ~= "windows" then
  local bee_output = bindir .. "/" .. (lm.os == "macos" and "libbee.dylib" or "libbee.so")
  lm:copy("copy_bee")({
    deps = { "bee" },
    inputs = { bindir .. "/bee.so" },
    outputs = { bee_output },
  })
  local output = lm.bindir
    .. "/"
    .. (lm.os == "windows" and "lua54.dll" or (lm.os == "macos" and "liblua54.dylib" or "liblua54.so"))

  lm:dll("lua54")({
    deps = { "source_lua54" },
  })

  lm:copy("copy_lua54")({
    deps = { "lua54" },
    inputs = { lm.bindir .. "/lua54.so" },
    outputs = { output },
  })
else
  if lm.os == "windows" then lm:dll("bee")({
    deps = { "source_bee", "lua54" },
  }) end
end

lm:phony("all")({
  deps = {
    "lua54",
    lm.os ~= "windows" and "copy_lua54",
    "bee",
    lm.os ~= "windows" and "copy_bee",
  },
})

lm:default({ "all" })
