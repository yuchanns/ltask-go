//go:build linux

package main

import _ "embed"

var luapattern = "*.so"

//go:embed build/bin/liblua54.so
var lualib []byte
