//go:build windows

package main

import _ "embed"

//go:embed build/.lua/lib/lua54.dll
var lualib []byte
