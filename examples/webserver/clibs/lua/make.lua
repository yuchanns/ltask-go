local lm = require("luamake")

if lm.os ~= "windows" then
  lm:source_set("source_lua54")({
    sources = {
      lm.base_dir .. "/bee/3rd/lua54/*.c",
      "!" .. lm.base_dir .. "/bee/3rd/lua54/onelua.c",
      "!" .. lm.base_dir .. "/bee/3rd/lua54/lua.c",
      "!" .. lm.base_dir .. "/bee/3rd/lua54/luac.c",
      "!" .. lm.base_dir .. "/bee/3rd/lua54/ltests.c",
    },
    includes = {
      lm.base_dir .. "/bee/3rd/lua54",
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
end
