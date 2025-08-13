package main

import (
	"os"

	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

func main() {
	fs, err := os.CreateTemp("", "")
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
	ltask.OpenLibs(L, lib)

	err = L.DoFile("main.lua")
	if err != nil {
		panic(err)
	}
}
