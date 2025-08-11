//go:build !windows

package ltask

import (
	"net"
	"os"
	"syscall"
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
	conn net.Conn
	file *os.File
}

func newConn(fd int) (*conn, error) {
	file := os.NewFile(uintptr(fd), "netfd")
	con, err := net.FileConn(file)
	return &conn{
		conn: con,
		file: file,
	}, err
}

func (c *conn) close() {
	c.conn.Close()
}

func (c *conn) write(b []byte) (n int, err error) {
	n, err = c.conn.Write(b)
	return
}

func (c *conn) read(b []byte) (n int, err error) {
	n, err = c.conn.Read(b)
	return
}
