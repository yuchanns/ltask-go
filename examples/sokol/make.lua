local lm = require("luamake")

lm.base_dir = lm:path(".")
lm.bindir = "build/bin"

lm:import("clibs/sokol/make.lua")
lm:import("clibs/lua/make.lua")

lm:dll("clibs")({
  deps = { "source_sokol", "source_lua54" },
  linux = {
    links = {
      "GL",
      "X11",
      "Xext",
      "Xi",
      "Xcursor",
      "Xrandr",
      "Xinerama",
      "dl",
      "pthread",
      "m",
    },
  },
  macos = {
    frameworks = {
      "Cocoa",
      "QuartzCore",
      "Metal",
      "MetalKit",
      "OpenGL",
      "AudioToolbox",
    },
  },
  windows = {
    links = {
      "kernel32",
      "user32",
      "shell32",
      "gdi32",
      "dxgi",
      "d3d11",
    },
  },
})

lm:phony("all")({
  deps = {
    "clibs",
  },
})

lm:default("all")
