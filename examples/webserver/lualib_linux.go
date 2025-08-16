//go:build linux

package main

import (
	_ "embed"
	"fmt"
	"os"
)

var libext = "so"
var libpattern = "*." + libext

//go:embed build/bin/liblua54.so
var lualib []byte

//go:embed build/bin/libbee.so
var beelib []byte

func installBee(tmpdir string) (err error) {
	fs, err := os.Create(fmt.Sprintf("%s/bee.%s", tmpdir, libext))
	if err != nil {
		return
	}
	defer fs.Close()
	_, err = fs.Write(beelib)
	return
}
