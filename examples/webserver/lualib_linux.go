//go:build linux

package main

import _ "embed"

var luapattern = "*.so"

//go:embed build/.lua/lib/liblua54.so
var lualib []byte
