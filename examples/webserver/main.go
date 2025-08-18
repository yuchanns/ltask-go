package main

import (
	"embed"
	"os"
	"path"
	"unsafe"

	"github.com/ebitengine/purego"
	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

//go:embed src/*.lua
var luafs embed.FS

var luaopenBeeSocket func(L unsafe.Pointer) int
var luaopenBeeEpoll func(L unsafe.Pointer) int

func luaopenlibs(L *lua.State) int {
	_ = L.GetGlobal("package")
	_, _ = L.GetField(-1, "preload")
	l := []*lua.Reg{
		{Name: "bee.socket", Func: func(L *lua.State) int {
			return luaopenBeeSocket(L.L())
		}},
		{Name: "bee.epoll", Func: func(L *lua.State) int {
			return luaopenBeeEpoll(L.L())
		}},
	}
	L.SetFuncs(l, 0)
	L.Pop(2)
	return 0
}

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
	_, err = fs.Write(clibs)
	if err != nil {
		panic(err)
	}
	err = fs.Close()
	if err != nil {
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

	clibs, err := loadLibrary(fs.Name())
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&luaopenBeeSocket, clibs, "luaopen_bee_socket")
	purego.RegisterLibFunc(&luaopenBeeEpoll, clibs, "luaopen_bee_epoll")

	ltask.UseEmbedFS(&luafs)
	ltask.OnServiceInit(luaopenlibs)
	ltask.OpenLibs(L, lib)
	luaopenlibs(L)

	scode, err := luafs.ReadFile("src/bootstrap.lua")
	if err != nil {
		panic(err)
	}
	err = L.DoString(string(scode))
	if err != nil {
		panic(err)
	}
}
