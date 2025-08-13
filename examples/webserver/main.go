package main

import (
	"embed"
	"os"

	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

//go:embed src/*.lua
var luafs embed.FS

func main() {
	fs, err := os.CreateTemp("", luapattern)
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

	defer os.Remove(fs.Name())

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

	scode, err := luafs.ReadFile("src/main.lua")
	if err != nil {
		panic(err)
	}
	err = L.DoString(string(scode))
	if err != nil {
		panic(err)
	}
}
