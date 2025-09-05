package main

import (
	"os"
	"runtime"

	"github.com/phuslu/log"
	"go.yuchanns.xyz/ltask/examples/sokol/internal/app"
	"go.yuchanns.xyz/lua"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	fs, err := os.CreateTemp("", libpattern)
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}
	_, err = fs.Write(clibs)
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}
	err = fs.Close()
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}
	defer os.Remove(fs.Name())

	lib, err := lua.New(fs.Name())
	if err != nil {
		log.Fatal().Msgf("%s", err)
	}
	defer lib.Close()

	app.New(lib).Run()
}
