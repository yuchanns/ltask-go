//go:build darwin

package main

import _ "embed"

var luapattern = "*.dylib"

//go:embed build/bin/liblua54.dylib
var lualib []byte
