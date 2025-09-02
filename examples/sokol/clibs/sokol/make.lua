local lm = require("luamake")

lm:source_set("source_sokol")({
  includes = {
    lm.base_dir .. "/3rd/sokol",
  },
  sources = {
    lm.os == "macos" and "./dummy.m" or "./dummy.c",
  },
  flags = {
    "-Wno-unknown-pragmas",
    "-fPIC",
  },
  visibility = "default",
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
