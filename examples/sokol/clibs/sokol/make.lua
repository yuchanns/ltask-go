local lm = require("luamake")

lm:source_set("source_sokol")({
  includes = {
    lm.base_dir .. "/3rd/sokol",
  },
  sources = {
    lm.os == "macos" and "./dummy.m" or "./dummy.c",
  },
  flags = {
    lm.os ~= "windows" and "-Wno-unknown-pragmas" or "-wd4068",
    lm.os ~= "windows" and "-fPIC",
  },
  msvc = {
    flags = {
      "/utf-8",
    },
  },
  visibility = "default",
  windows = {
    defines = {
      "SOKOL_D3D11",
      "SOKOL_DLL",
    },
  },
  macos = {
    defines = {
      "SOKOL_METAL",
    },
  },
  linux = {
    defines = {
      "SOKOL_GLCORE",
    },
  },
  defines = {
    "SOKOL_IMPL",
    "SOKOL_NO_ENTRY",
  },
})
