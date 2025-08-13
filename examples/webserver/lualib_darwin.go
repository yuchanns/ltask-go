//go:build darwin

package main

import _ "embed"

var luapattern = "*.dylib"

//go:embed build/.lua/lib/liblua54.dylib
var lualib []byte
