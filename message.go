package ltask

import "unsafe"

type session = uint64

type message struct {
	from    serviceId
	to      serviceId
	session session
	typ     int
	msg     unsafe.Pointer
	sz      int64
}

const (
	messageReceiptNone     = 0
	messageReceiptDone     = 1
	messageReceiptError    = 2
	messageReceiptBlock    = 3
	messageReceiptResponse = 4
)
