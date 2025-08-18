//go:build linux

package main

import (
	_ "embed"

	"github.com/ebitengine/purego"
)

var libpattern = "*.so"

//go:embed build/bin/clibs.so
var clibs []byte

func loadLibrary(path string) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
}
