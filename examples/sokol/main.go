package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
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

	L, err := lib.NewState()
	if err != nil {
		panic(err)
	}
	defer L.Close()

	ffi := loadFFI(L.FFI().Lib())

	L.OpenLibs()
	ltask.OpenLibs(L, lib)

	var desc = sappDesc{
		initCb:    purego.NewCallback(initCb),
		frameCb:   purego.NewCallback(frameCb),
		cleanupCb: purego.NewCallback(cleanupCb),
		eventCb:   purego.NewCallback(eventCb),
	}
	ffi.SappRun(unsafe.Pointer(&desc))
}

func init() {
	runtime.LockOSThread()
}

func initCb() {
	fmt.Println("init")
}

func frameCb() {
	fmt.Println("frame")
	time.Sleep(time.Second)
}

func cleanupCb() {
	fmt.Println("cleanup")
}

func eventCb(ev unsafe.Pointer) {
	fmt.Println("event", ev)
}
