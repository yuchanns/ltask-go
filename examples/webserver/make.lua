local lm = require("luamake")

lm.base_dir = lm:path(".")
lm.lua = "54"
lm.luadir = lm.base_dir .. "/3rd/bee.lua/3rd/lua" .. lm.lua

lm.bindir = "build/bin"

local function macos_version()
  local cxx = lm.cxx or "c++17"
  local version = cxx:match("^c%+%+(.+)$")
  if version == "17" then
    return "macos10.15"
  else
    return "macos13.3"
  end
end

lm:conf({
  compile_commands = "build",
  mode = "debug",
  c = "c17",
  cxx = "c++17",
  visibility = "default",
  windows = {
    defines = {
      "_CRT_SECURE_NO_WARNINGS",
      "_WIN32_WINNT=0x0602",
    },
  },
  msvc = {
    flags = "/utf-8",
    ldflags = lm.mode == "debug" and lm.arch == "x86_64" and {
      "/STACK:" .. 0x160000,
    },
  },
  macos = {
    flags = "-Wunguarded-availability",
    sys = macos_version(),
  },
  linux = {
    crt = "static",
    flags = "-fPIC",
    ldflags = {
      "-Wl,-E",
      "-static-libgcc",
    },
  },
})

lm:import("clibs/lua/make.lua")
lm:import("clibs/bee/make.lua")

lm:dll("clibs")({
  deps = { "source_lua54", lm.os == "windows" and "bee_utf8_crt", "source_bee" },
})

lm:copy("copy_clibs")({
  deps = { "clibs" },
  inputs = { lm.bindir .. "/clibs.so" },
  outputs = { lm.bindir .. "/clibs.dylib" },
})

lm:phony("all")({
  deps = {
    "clibs",
    lm.os == "macos" and "copy_clibs",
  },
})

lm:default({ "all" })
