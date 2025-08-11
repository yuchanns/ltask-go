//go:build windows

package ltask

import (
	"net"
	"unsafe"

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

var (
	ws2_32DLL = windows.NewLazySystemDLL("ws2_32.dll")
	procSend  = ws2_32DLL.NewProc("send")
	procRecv  = ws2_32DLL.NewProc("recv")
	procClose = ws2_32DLL.NewProc("closesocket")
)

func send(fd int, buf []byte, flags int) (n int, err error) {
	r1, _, err := procSend.Call(
		uintptr(fd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(flags),
	)
	if err != nil {
		return
	}
	n = int(r1)
	return
}

func recv(fd int, buf []byte, flags int) (n int, err error) {
	r1, _, err := procRecv.Call(
		uintptr(fd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(flags),
	)
	if err != nil {
		return
	}
	n = int(r1)
	return
}

func closeSocket(fd int) (err error) {
	r1, _, err := procClose.Call(uintptr(fd))
	if r1 != 0 {
		return err
	}
	return nil
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
	closeSocket(int(c.handle))
}

func (c *conn) write(b []byte) (n int, err error) {
	n, err = send(int(c.handle), b, 0)
	return
}

func (c *conn) read(b []byte) (n int, err error) {
	n, err = recv(int(c.handle), b, 0)
	if err == windows.ERROR_MORE_DATA {
		err = nil
	}
	return
}
