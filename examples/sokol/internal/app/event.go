package app

import (
	"go.yuchanns.xyz/ltask/examples/sokol/internal/sokol"
)

type eventMessage struct {
	typ string
	p1  int32
	p2  int32
}

func (em *eventMessage) toMouse(ev *sokol.SappEvent) {
	switch ev.Type {
	case sokol.SappEventTypeMouseMove:
		em.typ = "mouse_move"
		em.p1 = int32(ev.MouseX)
		em.p2 = int32(ev.MouseY)
	case sokol.SappEventTypeMouseDown, sokol.SappEventTypeMouseUp:
		em.typ = "mouse_button"
		em.p1 = int32(ev.MouseButton)
		if ev.Type == sokol.SappEventTypeMouseDown {
			em.p2 = 1
		} else {
			em.p2 = 0
		}
	case sokol.SappEventTypeMouseScroll:
		em.typ = "mouse_scroll"
		em.p1 = int32(ev.ScrollX)
		em.p2 = int32(ev.ScrollY)
	default:
		em.typ = "mouse"
		em.p1 = int32(ev.Type)
	}
}

func (em *eventMessage) toWindows(ev *sokol.SappEvent) {
	switch ev.Type {
	case sokol.SappEventTypeResized:
		em.typ = "window_resize"
		em.p1 = int32(ev.WindowWidth)
		em.p2 = int32(ev.WindowHeight)
	default:
		em.typ = "window"
		em.p1 = int32(ev.Type)
	}
}

func (em *eventMessage) toKey(ev *sokol.SappEvent) {
	switch ev.Type {
	case sokol.SappEventTypeChar:
		em.typ = "char"
		em.p1 = int32(ev.CharCode)
	default:
		em.typ = "key"
		em.p1 = int32(ev.Type)
	}
}

func eventUnpack(ev *sokol.SappEvent) (em *eventMessage) {
	em = &eventMessage{}
	switch ev.Type {
	case sokol.SappEventTypeMouseDown,
		sokol.SappEventTypeMouseUp,
		sokol.SappEventTypeMouseMove,
		sokol.SappEventTypeMouseEnter,
		sokol.SappEventTypeMouseLeave,
		sokol.SappEventTypeMouseScroll:
		// mouse message
		em.toMouse(ev)
		return
	case sokol.SappEventTypeResized:
		// window message
		em.toWindows(ev)
		return
	case sokol.SappEventTypeChar:
		// key message
		em.toKey(ev)
		return
	}
	em.typ = "message"
	em.p1 = int32(ev.Type)
	return
}
