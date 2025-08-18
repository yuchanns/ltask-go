//go:build darwin

package main

import (
	_ "embed"

	"github.com/ebitengine/purego"
)

var libpattern = "*.dylib"

//go:embed build/bin/clibs.dylib
var clibs []byte

func loadLibrary(path string) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
}
