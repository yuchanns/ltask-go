local lm = require("luamake")

local bindir = "build/bin"

lm:dll("lua54")({
  sources = { "lua54/*.c", "!lua54/onelua.c", "!lua54/lua.c", "!lua54/luac.c", "!lua54/ltests.c" },
  includes = {
    "lua54",
  },
  bindir = bindir,

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

if lm.os == "windows" then
  lm:default({ "lua54" })
else
  lm:copy("copy")({
    deps = { "lua54" },
    inputs = { bindir .. "/lua54.so" },
    outputs = { output },
  })

  lm:default({ "lua54", "copy" })
end
