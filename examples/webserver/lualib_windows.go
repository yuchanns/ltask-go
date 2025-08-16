//go:build windows

package main

import _ "embed"

var libext = "dll"
var libpattern = "*." + libext

//go:embed build/bin/lua54.dll
var lualib []byte

func installBee(_ string) (err error) {
	return
}
