package main

import (
	"reflect"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
	"go.yuchanns.xyz/lua"
)

type luaopenLib func(L unsafe.Pointer) int

type externalFFI struct {
	BeeSocket luaopenLib `ffi:"luaopen_bee_socket"`
	BeeEpoll  luaopenLib `ffi:"luaopen_bee_epoll"`
}

func externalOpenLibs(L *lua.State) {
	luaLOpenLibs := L.FFI().LuaLOpenlibs

	var effi externalFFI
	t := reflect.TypeOf(&effi).Elem()
	v := reflect.ValueOf(&effi).Elem()

	var l = []*lua.Reg{
		{Name: "json", Func: luaOpenJSON},
	}
	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Kind() != reflect.Func {
			continue
		}
		fname := field.Tag.Get("ffi")
		if fname == "" {
			continue
		}
		if _, ok := v.Field(i).Interface().(luaopenLib); !ok {
			continue
		}
		fptr := v.Field(i).Addr().Interface()
		purego.RegisterLibFunc(fptr, L.FFI().Lib(), fname)
		fn := *fptr.(*luaopenLib)
		l = append(l, &lua.Reg{
			Name: strings.ReplaceAll(strings.TrimPrefix(fname, "luaopen_"), "_", "."),
			Func: func(L *lua.State) int { return fn(L.L()) },
		})
	}

	clone := L.Clone

	L.FFI().LuaLOpenlibs = func(luaL unsafe.Pointer) {
		luaLOpenLibs(luaL)
		L := clone(luaL)
		L.GetGlobal("package")
		_ = L.GetField(-1, "preload")
		L.SetFuncs(l, 0)
		L.Pop(2)
	}
}
