//go:build windows

package ltask

import (
	"golang.org/x/sys/windows"
)

func fdDup(fd int) (int, error) {
	var newfd windows.Handle
	currentProc := windows.CurrentProcess()
	err := windows.DuplicateHandle(
		currentProc,
		windows.Handle(fd),
		currentProc,
		&newfd,
		0,
		true,
		windows.DUPLICATE_SAME_ACCESS,
	)
	if err != nil {
		return 0, err
	}
	return int(newfd), nil
}
