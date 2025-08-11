//go:build !windows

package ltask

import "syscall"

func fdDup(fd int) (int, error) {
	return syscall.Dup(fd)
}
