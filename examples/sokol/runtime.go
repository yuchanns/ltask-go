package main

import (
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego"
	"go.yuchanns.xyz/lua"
)

type sappDesc struct {
	initCb    uintptr
	frameCb   uintptr
	cleanupCb uintptr
	eventCb   uintptr

	_ [1000]byte
}

type externalFFI struct {
	Lib *lua.Lib

	SappRun       func(unsafe.Pointer) `ffi:"sapp_run"`
	SappQuit      func()               `ffi:"sapp_quit"`
	SargsShutdown func()               `ffi:"sargs_shutdown"`
}

var effi externalFFI

func loadExternalFFI(lib *lua.Lib) {
	t := reflect.TypeOf(&effi).Elem()
	v := reflect.ValueOf(&effi).Elem()
	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Kind() != reflect.Func {
			continue
		}
		fname := field.Tag.Get("ffi")
		if fname == "" {
			continue
		}
		fptr := v.Field(i).Addr().Interface()
		purego.RegisterLibFunc(fptr, lib.FFI().Lib(), fname)
	}
	effi.Lib = lib
}
