package main

import (
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego"
)

type sappDesc struct {
	initCb    uintptr
	frameCb   uintptr
	cleanupCb uintptr
	eventCb   uintptr

	_ [1000]byte
}

type externalFFI struct {
	SappRun func(unsafe.Pointer) `ffi:"sapp_run"`
}

func loadFFI(lib uintptr) *externalFFI {
	var effi externalFFI
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
		purego.RegisterLibFunc(fptr, lib, fname)
	}
	return &effi
}
