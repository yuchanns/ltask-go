package sokol

import (
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego"
	"go.yuchanns.xyz/lua"
)

type callback func() int
type eventCallback func(unsafe.Pointer) int

type AppContext interface {
	OnInit(app *App) (ret int)
	OnFrame(app *App) (ret int)
	OnCleanup(app *App) (ret int)
	OnEvent(app *App, ev unsafe.Pointer) (ret int)
}

type ffi struct {
	SappRun        func(unsafe.Pointer) `ffi:"sapp_run"`
	SappQuit       func()               `ffi:"sapp_quit"`
	SappFrameCount func() uint64        `ffi:"sapp_frame_count"`
}

type App struct {
	ctx AppContext

	ffi *ffi
}

func NewApp(ctx AppContext) *App {
	var ffi ffi
	t := reflect.TypeOf(&ffi).Elem()
	v := reflect.ValueOf(&ffi).Elem()
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
		purego.RegisterLibFunc(fptr, lua.FFI().Lib(), fname)
	}

	return &App{
		ctx: ctx,
		ffi: &ffi,
	}
}

func (app *App) Quit() {
	app.ffi.SappQuit()
}

func (app *App) FrameCount() uint64 {
	return app.ffi.SappFrameCount()
}

type sappDesc struct {
	initCb    uintptr
	frameCb   uintptr
	cleanupCb uintptr
	eventCb   uintptr

	_ [1000]byte
}

func (app *App) Run() {
	var desc = sappDesc{
		initCb: purego.NewCallback(func() int {
			return app.ctx.OnInit(app)
		}),
		frameCb: purego.NewCallback(func() int {
			return app.ctx.OnFrame(app)
		}),
		cleanupCb: purego.NewCallback(func() int {
			return app.ctx.OnCleanup(app)
		}),
		eventCb: purego.NewCallback(func(ev unsafe.Pointer) int {
			return app.ctx.OnEvent(app, ev)
		}),
	}
	app.ffi.SappRun(unsafe.Pointer(&desc))
}
