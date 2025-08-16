package main

import (
	"embed"
	"fmt"
	"os"
	"path"
	"strings"

	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

//go:embed src/*.lua
var luafs embed.FS

func main() {
	tmpdir := path.Join(os.TempDir(), "ltask")
	err := os.MkdirAll(tmpdir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpdir)

	fs, err := os.CreateTemp(tmpdir, libpattern)
	if err != nil {
		panic(err)
	}
	_, err = fs.Write(lualib)
	if err != nil {
		panic(err)
	}
	err = fs.Close()
	if err != nil {
		panic(err)
	}

	if err := installBee(tmpdir); err != nil {
		panic(err)
	}

	lib, err := lua.New(fs.Name())
	if err != nil {
		panic(err)
	}
	defer lib.Close()

	L, err := lib.NewState()
	if err != nil {
		panic(err)
	}
	defer L.Close()

	L.OpenLibs()
	ltask.UseEmbedFS(&luafs)
	ltask.OpenLibs(L, lib)

	cpath := strings.ReplaceAll(path.Join(tmpdir, fmt.Sprintf("?.%s", libext)), "\\", "\\\\")
	err = L.DoString(fmt.Sprintf("package.cpath = package.cpath .. ';%s'", cpath))
	if err != nil {
		panic(err)
	}
	scode, err := luafs.ReadFile("src/bootstrap.lua")
	if err != nil {
		panic(err)
	}
	err = L.DoString(string(scode))
	if err != nil {
		panic(err)
	}
}
