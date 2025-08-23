package ltask

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/phuslu/log"
	"go.yuchanns.xyz/lua"
)

func init() {
	log.DefaultLogger = log.Logger{
		Level:      log.DebugLevel,
		Caller:     1,
		TimeFormat: "2006-01-02 15:04:05.00",
		Writer: &log.ConsoleWriter{
			ColorOutput: false,
			Formatter: func(w io.Writer, a *log.FormatterArgs) (n int, err error) {
				return fmt.Fprintf(w, "[%s][%s]( %s ) %s\n%s", a.Time, strings.ToUpper(a.Level), a.Caller, a.Message, a.Stack)
			},
			Writer: os.Stderr,
		},
	}
}

func OpenLibs(L *lua.State, lib *lua.Lib) {
	_ = L.GetGlobal("package")
	_, _ = L.GetField(-1, "preload")

	L.PushLightUserData(lib)

	l := []*lua.Reg{
		{Name: "ltask.bootstrap", Func: ltaskBootstrapOpen},
	}
	L.SetFuncs(l, 1)
	L.Pop(2)
}

type serviceUd struct {
	task *ltask
	id   serviceId
}

const (
	threadNone = -1
)
