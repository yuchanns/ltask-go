package main

import (
	"embed"
	"os"

	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

//go:embed src/*.lua src/**/*.lua
var luafs embed.FS

func main() {
	fs, err := os.CreateTemp("", libpattern)
	if err != nil {
		panic(err)
	}
	_, err = fs.Write(clibs)
	if err != nil {
		panic(err)
	}
	err = fs.Close()
	if err != nil {
		panic(err)
	}
	defer os.Remove(fs.Name())

	err = lua.Init(fs.Name())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = lua.Deinit()
	}()

	L := lua.NewState()
	if err != nil {
		panic(err)
	}
	defer L.Close()

	externalOpenLibs(L)

	L.OpenLibs()

	ltask.UseEmbedFS(&luafs)
	ltask.OpenLibs(L)

	scode, err := luafs.ReadFile("src/bootstrap.lua")
	if err != nil {
		panic(err)
	}
	err = L.DoString(string(scode))
	if err != nil {
		panic(err)
	}
}
