package ltask

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"
	"unsafe"

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

type serviceUd struct {
	task *ltask
	id   serviceId
}

const (
	threadNone = -1
)

func getS(L *lua.State) *serviceUd {
	ud := L.ToUserData(L.UpValueIndex(1))
	if ud == nil {
		panic("Invalid service userdata")
	}
	return (*serviceUd)(ud)
}

var ltaskSleep = lua.NewCallback(func(L *lua.State) int {
	csec := L.OptInteger(1, 0)
	time.Sleep(centisecond * time.Duration(csec))
	return 0
})

var ltaskEventWait = lua.NewCallback(func(L *lua.State) int {
	event := (*sockEvent)(L.ToUserData(L.UpValueIndex(1)))
	r := event.wait()
	L.PushBoolean(r > 0)
	return 1
})

var ltaskEventInit = lua.NewCallback(func(L *lua.State) int {
	s := getS(L)
	index := s.task.services.getSockevent(s.id)
	if index >= 0 {
		return L.Errorf("Already init event")
	}
	index = int64(s.task.allocSockevent())
	if index < 0 {
		return L.Errorf("Too many sockevents")
	}
	event := &s.task.event[index]
	L.PushLightUserData(event)
	L.PushCClousure(ltaskEventWait, 1)
	if !event.open() {
		return L.Errorf("Create sockevent fail")
	}
	s.task.services.initSockevent(s.id, index)
	fd := event.fd()
	L.PushLightUserData(*(*unsafe.Pointer)(unsafe.Pointer(&fd)))
	return 2
})

var ltaskTimerAdd = lua.NewCallback(func(L *lua.State) int {
	s := getS(L)
	t := s.task.timer
	if t == nil {
		return L.Errorf("Init timer before bootstrap")
	}
	ev := &timerEvent{
		session: int32(L.CheckInteger(1)),
		id:      s.id,
	}
	ti := L.CheckInteger(2)
	if ti < 0 || ti != int64(int32(ti)) {
		return L.Errorf("Invalid timer time: %d", ti)
	}
	t.Add(ev, time.Duration(ti)*centisecond)
	return 0
})

var ltaskTimerUpdate = lua.NewCallback(func(L *lua.State) int {
	s := getS(L)
	t := s.task.timer
	if t == nil {
		return L.Errorf("Init timer before bootstrap")
	}
	if L.GetTop() > 1 {
		L.SetTop(1)
		L.CheckType(1, lua.LUA_TTABLE)
	}
	var idx int64
	t.Update(func(event *timerEvent) {
		v := int64(event.session)<<32 | int64(event.id)
		L.PushInteger(v)
		idx++
		L.SetI(1, int64(idx))
	})
	n := int64(L.RawLen(1))
	for i := int64(idx + 1); i <= n; i++ {
		L.PushNil()
		L.SetI(1, i)
	}
	return 1
})

var ltaskNow = lua.NewCallback(func(L *lua.State) int {
	s := getS(L)
	t := s.task.timer
	if t == nil {
		return L.Errorf("Init timer before bootstrap")
	}
	start := t.Start()
	now := t.Now()
	L.PushInteger(start + now/100)
	L.PushInteger(start*100 + now)
	return 2
})

var lself = lua.NewCallback(func(L *lua.State) int {
	s := getS(L)
	L.PushInteger(int64(s.id))
	return 1
})

var lworkerId = lua.NewCallback(func(L *lua.State) int {
	s := getS(L)
	worker := s.task.getWorkerId(s.id)
	if worker >= 0 {
		L.PushInteger(int64(worker))
		return 1
	}
	return 0
})

var lworkerBind = lua.NewCallback(func(L *lua.State) int {
	s := getS(L)
	if L.IsNoneOrNil(1) {
		s.task.services.setBindingThread(s.id, threadNone)
		return 0
	}
	worker := L.CheckInteger(1)
	if worker < 0 || worker >= int64(len(s.task.workers)) {
		return L.Errorf("Invalid worker id: %d", worker)
	}
	s.task.services.setBindingThread(s.id, int32(worker))
	return 0
})

var ltaskLabel = lua.NewCallback(func(L *lua.State) int {
	s := getS(L)
	label := s.task.services.getLabel(s.id)
	L.PushString(label)
	return 1
})

var ltaskPushLog = lua.NewCallback(func(L *lua.State) int {
	L.CheckType(1, lua.LUA_TLIGHTUSERDATA)
	data := L.ToUserData(1)
	sz := L.CheckInteger(2)
	s := getS(L)
	if !s.task.pushLog(s.id, data, sz) {
		return L.Errorf("log error")
	}

	return 0
})

var ltaskPopLog = lua.NewCallback(func(L *lua.State) int {
	s := getS(L)
	m, ok := s.task.lqueue.pop()
	if !ok {
		return 0
	}
	start := int64(0)
	t := s.task.timer
	if t != nil {
		start = t.Start() * 100
	}
	L.PushInteger(m.timestamp + start)
	L.PushInteger(int64(m.id))
	L.PushLightUserData(m.msg)
	L.PushInteger(m.sz)
	return 4
})

var ltaskReadFile = lua.NewCallback(func(L *lua.State) int {
	fileName := L.CheckString(1)
	var (
		err   error
		scode []byte
	)
	for _, embedfs := range embedfsList {
		scode, err = embedfs.ReadFile(fileName)
		if err == nil {
			L.PushString(string(scode))
			return 1
		}
		if !os.IsNotExist(err) {
			break
		}
	}
	L.PushNil()
	L.PushString(err.Error())
	return 2
})

var ltaskDoFile = lua.NewCallback(func(L *lua.State) int {
	fileName := L.CheckString(1)
	var (
		err   error
		scode []byte
	)
	for _, embedfs := range embedfsList {
		scode, err = embedfs.ReadFile(fileName)
		if err == nil {
			L.SetTop(0)
			err = L.DoString(string(scode))
			if err != nil {
				L.PushNil()
				L.PushString(err.Error())
				return 2
			}
			return L.GetTop()
		}
		if !os.IsNotExist(err) {
			break
		}
	}
	L.PushNil()
	L.PushString(err.Error())
	return 2
})

var ltaskLoadFile = lua.NewCallback(func(L *lua.State) int {
	fileName := L.CheckString(1)
	var (
		err   error
		scode []byte
	)
	for _, embedfs := range embedfsList {
		scode, err = embedfs.ReadFile(fileName)
		if err == nil {
			err = L.LoadString(string(scode))
			if err != nil {
				L.PushNil()
				L.PushString(err.Error())
				return 2
			}
			return 1
		}
		if !os.IsNotExist(err) {
			break
		}
	}
	L.PushNil()
	L.PushString(err.Error())
	return 2
})

var ltaskSearchPath = lua.NewCallback(func(L *lua.State) int {
	name := L.CheckString(1)
	pattern := L.CheckString(2)

	patterns := strings.Split(pattern, ";")

	modulePath := strings.ReplaceAll(name, ".", "/")

	for _, pat := range patterns {
		fullPattern := strings.ReplaceAll(pat, "?", modulePath)

		dir := path.Dir(fullPattern)
		base := path.Base(fullPattern)

		for _, embedfs := range embedfsList {
			entries, err := embedfs.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if entry.Name() == base {
					L.PushString(path.Join(dir, entry.Name()))
					return 1
				}
			}
		}
	}
	return 0
})
