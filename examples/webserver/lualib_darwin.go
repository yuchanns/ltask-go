//go:build darwin

package main

import (
	_ "embed"
)

var libpattern = "*.dylib"

//go:embed build/bin/clibs.dylib
var clibs []byte
