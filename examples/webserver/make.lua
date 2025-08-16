local lm = require("luamake")

local bindir = "build/bin"
lm.bindir = bindir

lm.EXE = "lua"
lm:import("bee/make.lua")

if lm.os ~= "windows" then
  lm:source_set("source_lua54")({
    sources = { "lua54/*.c", "!lua54/onelua.c", "!lua54/lua.c", "!lua54/luac.c", "!lua54/ltests.c" },
    includes = {
      "lua54",
    },

    visibility = "default",
    links = {
      "m",
    },

    windows = {
      defines = {
        "LUA_BUILD_AS_DLL",
      },
    },

    macos = {
      defines = {
        "LUA_USE_MACOSX",
      },
    },

    linux = {
      defines = {
        "LUA_USE_LINUX",
      },
    },

    gcc = {
      flags = {
        "-fPIC",
      },
    },

    clang = {
      flags = {
        "-fPIC",
      },
    },
  })

  local output = bindir
    .. "/"
    .. (lm.os == "windows" and "lua54.dll" or (lm.os == "macos" and "liblua54.dylib" or "liblua54.so"))

  lm:dll("lua54")({
    deps = { "source_lua54" },
  })

  lm:copy("copy_lua54")({
    deps = { "lua54" },
    inputs = { bindir .. "/lua54.so" },
    outputs = { output },
  })
  local bee_output = bindir .. "/" .. (lm.os == "macos" and "libbee.dylib" or "libbee.so")
  lm:copy("copy_bee")({
    deps = { "bee" },
    inputs = { bindir .. "/bee.so" },
    outputs = { bee_output },
  })
end

lm:phony("all")({
  deps = {
    "lua54",
    lm.os ~= "windows" and "copy_lua54",
    lm.os ~= "windows" and "bee",
    lm.os ~= "windows" and "copy_bee",
  },
})

lm:default({ "all" })
