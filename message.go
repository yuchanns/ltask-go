package ltask

type session uint64

type message struct {
	from    serviceId
	to      serviceId
	session session
	typ     int
	msg     any
	sz      int64
}
