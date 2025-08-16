//go:build windows

package main

import (
	_ "embed"
	"fmt"
	"os"
	"path"
)

var libext = "dll"
var libpattern = "*." + libext

//go:embed build/bin/lua54.dll
var lualib []byte

//go:embed build/bin/bee.dll
var beelib []byte

func installBee(tmpdir string) (err error) {
	lib := path.Join(tmpdir, fmt.Sprintf("bee.%s", libext))
	fs, err := os.Create(lib)
	if err != nil {
		return
	}
	defer fs.Close()
	_, err = fs.Write(beelib)
	return
}
