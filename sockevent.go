package ltask

import (
	"net"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/phuslu/log"
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
		log.Debug().Msgf("sockEvent: dial %s failed: %v", addr.String(), err)
		return
	}
	defer writeConn.Close()
	runtime.SetFinalizer(writeConn, nil)

	var readConn net.Conn
	select {
	case readConn = <-connChan:
		defer readConn.Close()
	case err = <-errChan:
		log.Debug().Msgf("sockEvent: accept %s failed: %v", addr.String(), err)
		return
	}

	if tcpConn, ok := writeConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(false)
		if file, err := tcpConn.File(); err == nil {
			defer file.Close()
			fd, _ := fdDup(int(file.Fd()))
			s.pipe[1] = fd
		}
	}
	if tcpConn, ok := readConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(false)
		if file, err := tcpConn.File(); err == nil {
			defer file.Close()
			fd, _ := fdDup(int(file.Fd()))
			s.pipe[0] = fd
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
	fd, _ := fdDup(s.pipe[1])
	file := os.NewFile(uintptr(fd), "netfd")
	defer file.Close()
	conn, err := net.FileConn(file)
	if err != nil {
		return
	}
	defer conn.Close()
	_, _ = conn.Write([]byte{0})
}

func (s *sockEvent) wait() (n int) {
	if s.pipe[0] == socketInvalid {
		return
	}
	fd, _ := fdDup(s.pipe[0])
	file := os.NewFile(uintptr(fd), "netfd")
	defer file.Close()
	conn, err := net.FileConn(file)
	if err != nil {
		return
	}
	defer conn.Close()
	n, err = conn.Read(make([]byte, 128))
	atomic.StoreInt32(&s.e, 0)
	return
}

func (s *sockEvent) fd() int {
	return s.pipe[0]
}
