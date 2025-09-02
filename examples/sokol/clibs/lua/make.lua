local lm = require("luamake")

lm.rootdir = lm.base_dir .. "/3rd/lua54"

lm:source_set("source_lua54")({
  sources = {
    lm.rootdir .. "/*.c",
    "!" .. lm.rootdir .. "/onelua.c",
    "!" .. lm.rootdir .. "/lua.c",
    "!" .. lm.rootdir .. "/luac.c",
    "!" .. lm.rootdir .. "/ltests.c",
  },
  includes = {
    lm.rootdir,
  },
  visibility = "default",
  links = {
    lm.os ~= "windows" and "m",
    lm.os ~= "windows" and "dl",
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
