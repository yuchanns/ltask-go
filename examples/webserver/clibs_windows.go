//go:build windows

package main

import (
	_ "embed"
)

var libpattern = "*.dll"

//go:embed build/bin/clibs.dll
var clibs []byte
