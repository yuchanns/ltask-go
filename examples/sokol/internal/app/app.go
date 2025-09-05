package app

import (
	"fmt"
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

func (ctx *Context) invokeCallback(idx int, nargs int) {
	ctx.L.PushValue(idx)
	if nargs > 0 {
		ctx.L.Insert(-nargs - 1)
	}
	if err := ctx.L.PCall(nargs, 0, 0); err != nil {
		log.Error().Msgf("error invoking callback: %v", err)
	}
}

func (ctx *Context) OnFrame(app *sokol.App) (ret int) {
	ctx.L.PushInteger(int64(app.FrameCount()))
	ctx.invokeCallback(FrameCallback, 1)
	return 0
}

func (ctx *Context) OnCleanup(app *sokol.App) (ret int) {
	ctx.invokeCallback(CleanupCallback, 0)
	return 0
}

func (ctx *Context) OnEvent(app *sokol.App, ev unsafe.Pointer) (ret int) {
	ctx.L.PushLightUserData(ev)
	ctx.invokeCallback(EventCallback, 1)
	return 0
}

const (
	FrameCallback   = 1
	EventCallback   = 2
	CleanupCallback = 3
)

func msgHandler(L *lua.State) int {
	msg := L.ToString(1)
	L.Traceback(L, msg, 1)
	return 1
}

func pmain(L *lua.State) int {
	externalOpenLibs(L)
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
	filename := args[0]
	if err := L.LoadString(fmt.Sprintf(`local func = loadfile("%s"); return func(...)`, filename)); err != nil {
		return L.Errorf("error load and running %s: %v", filename, err)
	}
	L.Insert(-argN - 1)
	if err := L.PCall(argN, 1, 0); err != nil {
		return L.Errorf("error running %s: %v", filename, err)
	}

	return 1
}

func (ctx *Context) start() (err error) {
	L, err := ctx.Lib.NewState()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			L.Close()
		}
	}()

	L.PushGoFunction(msgHandler)
	L.PushGoFunction(pmain)
	err = L.PCall(0, 1, 1)
	if err != nil {
		return
	}

	ctx.L = L

	err = ctx.initCallback()
	return
}

type callbackFn struct {
	name string
	idx  int
}

func (cb callbackFn) install(L *lua.State) (err error) {
	if L.GetField(-1, cb.name) != lua.LUA_TFUNCTION {
		err = fmt.Errorf("missing function: %s", cb.name)
		return
	}
	L.Insert(cb.idx)
	return
}

func (ctx *Context) initCallback() (err error) {
	L := ctx.L
	if L.Type(-1) != lua.LUA_TTABLE {
		err = fmt.Errorf("error running pmain: must return a table")
		return
	}

	callbackFns := []callbackFn{
		{name: "frame", idx: FrameCallback},
		{name: "event", idx: EventCallback},
		{name: "cleanup", idx: CleanupCallback},
	}
	for _, cb := range callbackFns {
		if err = cb.install(L); err != nil {
			return
		}
	}
	L.SetTop(len(callbackFns))
	return
}
