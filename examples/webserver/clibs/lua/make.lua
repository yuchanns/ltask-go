local lm = require("luamake")

lm:source_set("source_lua54")({
  sources = {
    lm.luadir .. "/*.c",
    lm.os == "windows" and lm.base_dir .. "/3rd/bee.lua/3rd/lua-patch/bee_assert.c",
    "!" .. lm.luadir .. "/onelua.c",
    "!" .. lm.luadir .. "/lua.c",
    "!" .. lm.luadir .. "/luac.c",
    "!" .. lm.luadir .. "/ltests.c",
  },
  includes = {
    lm.luadir,
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

  msvc = lm.fast_setjmp ~= "off" and {
    defines = "BEE_FAST_SETJMP",
    sources = ("3rd/lua-patch/fast_setjmp_%s.s"):format(lm.arch),
  },
})
