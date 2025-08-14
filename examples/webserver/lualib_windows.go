//go:build windows

package main

import _ "embed"

var luapattern = "*.dll"

//go:embed build/bin/lua54.dll
var lualib []byte
