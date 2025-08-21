//go:build !windows

package main

import (
	_ "embed"
)

var libpattern = "*.so"

//go:embed build/bin/clibs.so
var clibs []byte
