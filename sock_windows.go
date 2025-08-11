//go:build windows

package ltask

import (
	"net"

	"golang.org/x/sys/windows"
)

func fdDup(fd int) (int, error) {
	pid := windows.GetCurrentProcessId()

	var info windows.WSAProtocolInfo
	err := windows.WSADuplicateSocket(windows.Handle(fd), uint32(pid), &info)
	if err != nil {
		return 0, err
	}

	newfd, err := windows.WSASocket(
		info.AddressFamily,
		info.SocketType,
		info.Protocol,
		&info,
		0,
		windows.WSA_FLAG_OVERLAPPED,
	)
	if err != nil {
		return 0, err
	}
	return int(newfd), nil
}

func fdGet(tcpConn *net.TCPConn) (fd int, err error) {
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return
	}
	rawConn.Control(func(rawfd uintptr) {
		fd, err = fdDup(int(rawfd))
	})
	return
}

type conn struct {
	handle windows.Handle
}

func newConn(handle int) (*conn, error) {
	return &conn{
		handle: windows.Handle(handle),
	}, nil
}

func (c *conn) close() {
	windows.Close(c.handle)
}

func (c *conn) write(b []byte) (n int, err error) {
	n, err = windows.Write(c.handle, b)
	return
}

func (c *conn) read(b []byte) (n int, err error) {
	n, err = windows.Read(c.handle, b)
	if err == windows.ERROR_MORE_DATA {
		err = nil
	}
	err = nil
	return
}
