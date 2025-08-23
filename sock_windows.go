//go:build windows

package ltask

import (
	"fmt"
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
	ws2_32DLL           = windows.NewLazySystemDLL("ws2_32.dll")
	procSend            = ws2_32DLL.NewProc("send")
	procRecv            = ws2_32DLL.NewProc("recv")
	procClose           = ws2_32DLL.NewProc("closesocket")
	procWSAGetLastError = ws2_32DLL.NewProc("WSAGetLastError")
)

func send(fd int, buf []byte, flags int) (n int, err error) {
	if len(buf) == 0 {
		return 0, nil
	}

	r1, _, _ := procSend.Call(
		uintptr(fd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(flags),
	)

	if r1 == uintptr(0xFFFFFFFF) { // SOCKET_ERROR
		errno, _, _ := procWSAGetLastError.Call()
		return 0, fmt.Errorf("send failed with error: %d", errno)
	}

	return int(r1), nil
}

func recv(fd int, buf []byte, flags int) (n int, err error) {
	if len(buf) == 0 {
		return 0, nil
	}

	r1, _, _ := procRecv.Call(
		uintptr(fd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(flags),
	)

	if r1 == uintptr(0xFFFFFFFF) { // SOCKET_ERROR
		errno, _, _ := procWSAGetLastError.Call()
		if errno == 10035 { // WSAEWOULDBLOCK
			return 0, nil
		}
		return 0, fmt.Errorf("recv failed with error: %d", errno)
	}

	return int(r1), nil
}

func closeSocket(fd int) error {
	r1, _, _ := procClose.Call(uintptr(fd))
	if r1 != 0 {
		errno, _, _ := procWSAGetLastError.Call()
		return fmt.Errorf("closesocket failed with error: %d", errno)
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

func newConn(handle int) *conn {
	return &conn{
		handle: windows.Handle(handle),
	}
}

func (c *conn) close() (err error) {
	return closeSocket(int(c.handle))
}

func (c *conn) write(b []byte) (n int, err error) {
	return send(int(c.handle), b, 0)
}

func (c *conn) read(b []byte) (n int, err error) {
	return recv(int(c.handle), b, 0)
}
