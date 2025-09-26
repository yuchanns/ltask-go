//go:build !windows

package ltask

import (
	"golang.org/x/sys/unix"
)

func bytePtrFromString(s string) (*byte, error) {
	if s == "" {
		return new(byte), nil
	}

	ptr, err := unix.BytePtrFromString(s)
	if err != nil {
		return nil, err
	}

	return ptr, nil
}

func bytePtrToString(p *byte) string {
	if p == nil {
		return ""
	}
	return unix.BytePtrToString(p)
}
