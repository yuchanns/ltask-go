package ltask

import (
	"net"
	"os"
	"runtime"
	"sync/atomic"
)

const (
	socketInvalid = -1
)

type sockEvent struct {
	pipe [2]int
	e    atomicInt
}

func (s *sockEvent) init() {
	s.pipe[0] = socketInvalid
	s.pipe[1] = socketInvalid

	atomic.StoreInt32(&s.e, 1)
}

func (s *sockEvent) open() (ok bool) {
	if s.pipe[0] != socketInvalid {
		return
	}
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return
		}
	}
	defer func() {
		listener.Close()
		if !ok {
			s.close()
		}
	}()

	addr := listener.Addr()

	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			errChan <- err
			return
		}
		connChan <- conn
	}()

	writeConn, err := net.Dial(listener.Addr().Network(), addr.String())
	if err != nil {
		return
	}
	runtime.SetFinalizer(writeConn, nil)

	var readConn net.Conn
	select {
	case readConn = <-connChan:
	case err = <-errChan:
		writeConn.Close()
		return
	}
	runtime.SetFinalizer(readConn, nil)

	if tcpConn, ok := writeConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(false)
		if file, err := tcpConn.File(); err == nil {
			s.pipe[1] = int(file.Fd())
		}
	}
	if tcpConn, ok := readConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(false)
		if file, err := tcpConn.File(); err == nil {
			s.pipe[0] = int(file.Fd())
		}
	}

	_, err = writeConn.Write([]byte{0})
	if err != nil {
		return
	}
	atomic.StoreInt32(&s.e, 0)
	ok = true
	return
}

func (s *sockEvent) close() {
	if s.pipe[0] != socketInvalid {
		file := os.NewFile(uintptr(s.pipe[0]), "netfd")
		conn, err := net.FileConn(file)
		if err != nil {
			return
		}
		conn.Close()
	}
	if s.pipe[1] != socketInvalid {
		file := os.NewFile(uintptr(s.pipe[1]), "netfd")
		conn, err := net.FileConn(file)
		if err != nil {
			return
		}
		conn.Close()
	}
}

func (s *sockEvent) trigger() {
	if s.pipe[1] == socketInvalid {
		return
	}
	if atomic.LoadInt32(&s.e) != 0 {
		return
	}
	atomic.StoreInt32(&s.e, 1)
	file := os.NewFile(uintptr(s.pipe[1]), "netfd")
	conn, err := net.FileConn(file)
	if err != nil {
		return
	}
	_, _ = conn.Write([]byte{0})
}

func (s *sockEvent) wait() (n int) {
	if s.pipe[0] == socketInvalid {
		return
	}
	file := os.NewFile(uintptr(s.pipe[0]), "netfd")
	conn, err := net.FileConn(file)
	if err != nil {
		return
	}
	n, err = conn.Read(make([]byte, 128))
	atomic.StoreInt32(&s.e, 0)
	return
}

func (s *sockEvent) fd() int {
	return s.pipe[0]
}
