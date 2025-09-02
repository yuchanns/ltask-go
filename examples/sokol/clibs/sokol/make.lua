local lm = require("luamake")

lm:source_set("source_sokol")({
  includes = {
    lm.base_dir .. "/3rd/sokol",
  },
  sources = {
    "./dummy.c",
  },
  visibility = "default",
  linux = {
    flags = {
      "-Wno-unknown-pragmas",
      "-fPIC",
    },
    defines = {
      "SOKOL_GLCORE",
    },
  },
  defines = {
    "SOKOL_IMPL",
    "SOKOL_NO_ENTRY",
  },
})
