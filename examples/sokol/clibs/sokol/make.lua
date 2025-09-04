local lm = require("luamake")

lm:source_set("source_sokol")({
  includes = {
    lm.base_dir .. "/3rd/sokol",
  },
  sources = {
    lm.os == "macos" and "./dummy.m" or "./dummy.c",
  },
  msvc = {
    flags = {
      "/utf-8",
      "-wd4068",
    },
  },
  gcc = {
    flags = { "-Wno-unknown-pragmas", "-fPIC" },
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
