//go:build !windows

package ltask

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func fdDup(fd int) (int, error) {
	return syscall.Dup(fd)
}

func fdGet(tcpConn *net.TCPConn) (fd int, err error) {
	file, err := tcpConn.File()
	if err != nil {
		return
	}
	defer file.Close()
	fd, err = fdDup(int(file.Fd()))
	return
}

type conn struct {
	fd int
}

func newConn(fd int) *conn {
	return &conn{
		fd: fd,
	}
}

func (c *conn) close() {
	unix.Close(c.fd)
}

func (c *conn) write(b []byte) (n int, err error) {
	n, err = unix.Write(c.fd, b)
	return
}

func (c *conn) read(b []byte) (n int, err error) {
	n, err = unix.Read(c.fd, b)
	return
}
