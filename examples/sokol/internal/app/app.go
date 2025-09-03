package app

import (
	"unsafe"

	"github.com/phuslu/log"
	"github.com/spf13/pflag"
	"go.yuchanns.xyz/ltask"
	"go.yuchanns.xyz/ltask/examples/sokol/internal/sokol"
	"go.yuchanns.xyz/lua"
)

type Context struct {
	L   *lua.State
	Lib *lua.Lib
}

func New(lib *lua.Lib) *sokol.App {
	return sokol.NewApp(lib.FFI().Lib(), &Context{Lib: lib})
}

func (ctx *Context) OnInit(app *sokol.App) (ret int) {
	pflag.Parse()
	if len(pflag.Args()) < 1 {
		log.Error().Msgf("Need startup filename\n")
		app.Quit()
		return
	}
	if err := ctx.start(); err != nil {
		log.Error().Msgf("Failed to start Lua: %v\n", err)
		app.Quit()
	}
	return
}

func (ctx *Context) OnFrame(app *sokol.App) (ret int) {
	log.Debug().Msg("frame")
	return 0
}

func (ctx *Context) OnCleanup(app *sokol.App) (ret int) {
	log.Debug().Msg("cleanup")
	return 0
}

func (ctx *Context) OnEvent(app *sokol.App, ev unsafe.Pointer) (ret int) {
	log.Debug().Msgf("event: %v", ev)
	return 0
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

func (ctx *Context) start() (err error) {
	L, err := ctx.Lib.NewState()
	if err != nil {
		return
	}
	ctx.L = L
	ctx.L.PushGoFunction(pmain)
	err = ctx.L.PCall(0, 0, 0)
	if err == nil {
		return
	}
	ctx.L.Close()
	return
}
