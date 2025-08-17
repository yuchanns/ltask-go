local lm = require("luamake")

lm.rootdir = lm.base_dir .. "/3rd/bee.lua"

lm:source_set("source_bee")({
  includes = lm.luadir,
  sources = "3rd/lua-seri/lua-seri.cpp",
  msvc = {
    flags = "/wd4244",
  },
})

lm:source_set("source_bee")({
  sources = "3rd/fmt/format.cc",
})

local OS = {
  "win",
  "posix",
  "osx",
  "linux",
  "bsd",
}

local function need(lst)
  local map = {}
  if type(lst) == "table" then
    for _, v in ipairs(lst) do
      map[v] = true
    end
  else
    map[lst] = true
  end
  local t = {}
  for _, v in ipairs(OS) do
    if not map[v] then
      t[#t + 1] = "!bee/**/*_" .. v .. ".cpp"
      t[#t + 1] = "!bee/" .. v .. "/**/*.cpp"
    end
  end
  return t
end

lm:source_set("source_bee")({
  includes = {
    ".",
    lm.luadir,
  },
  sources = "bee/**/*.cpp",
  msvc = lm.analyze and {
    flags = "/analyze",
  },
  gcc = lm.analyze and {
    flags = {
      "-fanalyzer",
      "-Wno-analyzer-use-of-uninitialized-value",
    },
  },
  windows = {
    sources = need("win"),
  },
  macos = {
    sources = {
      need({
        "osx",
        "posix",
      }),
    },
  },
  ios = {
    sources = {
      "!bee/filewatch/**/",
      need({
        "osx",
        "posix",
      }),
    },
  },
  linux = {
    sources = need({
      "linux",
      "posix",
    }),
  },
})

lm:source_set("source_bee")({
  includes = {
    ".",
    lm.luadir,
  },
  sources = {
    "binding/*.cpp",
    "3rd/lua-patch/bee_newstate.c",
  },
  msvc = lm.analyze and {
    flags = "/analyze",
  },
  gcc = lm.analyze and {
    flags = {
      "-fanalyzer",
      "-Wno-analyzer-use-of-uninitialized-value",
    },
  },
  windows = {
    defines = "_CRT_SECURE_NO_WARNINGS",
    sources = {
      "binding/port/lua_windows.cpp",
    },
    links = {
      "ntdll",
      "ws2_32",
      "ole32",
      "user32",
      "version",
      "synchronization",
      lm.arch == "x86" and "dbghelp",
    },
  },
  mingw = {
    links = {
      "uuid",
      "stdc++fs",
    },
  },
  linux = {
    ldflags = "-pthread",
    links = {
      "stdc++fs",
      "unwind",
      "bfd",
    },
  },
  macos = {
    frameworks = {
      "Foundation",
      "CoreFoundation",
      "CoreServices",
    },
  },
})

if lm.os == "windows" then
  lm:source_set("bee_utf8_crt")({
    includes = ".",
    sources = "3rd/lua-patch/bee_utf8_crt.cpp",
  })
end
