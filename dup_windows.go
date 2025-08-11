//go:build windows

package ltask

import (
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
