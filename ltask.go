package ltask

import (
	"runtime"

	"go.yuchanns.xyz/lua/lua54"
)

var lib *lua.Lib

func init() {
	var path string
	switch runtime.GOOS {
	case "windows":
		path = "./lua/lua54/.lua/lib/lua54.dll"
	case "linux":
		path = "./lua/lua54/.lua/lib/liblua54.so"
	case "darwin":
		path = "./lua/lua54/.lua/lib/liblua54.dylib"
	}
	var err error
	lib, err = lua.New(path)
	if err != nil {
		panic(err)
	}
}

func Hello() (err error) {
	L, err := lib.NewState()
	if err != nil {
		return
	}
	L.OpenLibs()
	err = L.DoString(`print("Hello from Lua!")`)
	return
}
