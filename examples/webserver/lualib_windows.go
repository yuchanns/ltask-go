//go:build windows

package main

import (
	_ "embed"

	"golang.org/x/sys/windows"
)

var libpattern = "*.dll"

//go:embed build/bin/clibs.dll
var clibs []byte

func loadLibrary(path string) (uintptr, error) {
	handle, err := windows.LoadLibrary(path)
	if err != nil {
		return 0, err
	}
	return uintptr(handle), nil
}
