package main

import (
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/spf13/pflag"
	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/lua"
)

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

	lib, err := lua.New(fs.Name())
	if err != nil {
		panic(err)
	}
	defer lib.Close()

	loadExternalFFI(lib)

	var desc = sappDesc{
		initCb:    purego.NewCallback(appInit),
		frameCb:   purego.NewCallback(frameCb),
		cleanupCb: purego.NewCallback(cleanupCb),
		eventCb:   purego.NewCallback(eventCb),
	}
	effi.SappRun(unsafe.Pointer(&desc))
}

func init() {
	runtime.LockOSThread()
}

type appContext struct {
	L *lua.State
}

func pmain(L *lua.State) int {
	L.OpenLibs()
	ltask.OpenLibs(L)
	args := pflag.Args()
	L.CheckStack(len(args) + 1)
	L.NewTable()
	argTableIdx := L.GetTop()
	for _, v := range args {
		L.PushString(v)
	}
	argN := L.GetTop() - argTableIdx + 1
	if err := L.LoadFile(args[0]); err != nil {
		return L.Errorf("cannot load %s: %v", args[0], err)
	}
	L.Insert(-argN - 1)
	if err := L.PCall(argN, 0, 0); err != nil {
		return L.Errorf("error running %s: %v", args[0], err)
	}

	return 0
}

func (ctx *appContext) start() {
	ctx.L.PushGoFunction(pmain)
	if err := ctx.L.PCall(0, 0, 0); err != nil {
		fmt.Fprintf(os.Stderr, "Error in pmain: %v\n", err)
		ctx.L.Close()
		effi.SappQuit()
		return
	}
}

var ctx *appContext

func appInit() (ret int) {
	pflag.Parse()
	if len(pflag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "Need startup filename\n")
		effi.SappQuit()
		return
	}
	L, err := effi.Lib.NewState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Lua state: %v\n", err)
		return
	}
	ctx = &appContext{L: L}
	ctx.start()
	effi.SargsShutdown()
	return
}

func frameCb() int {
	// fmt.Println("frame")
	return 0
}

func cleanupCb() int {
	fmt.Println("cleanup")
	return 0
}

func eventCb(ev unsafe.Pointer) int {
	// fmt.Println("event", ev)
	return 0
}
