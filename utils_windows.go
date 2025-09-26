//go:build windows

package ltask

import (
	"golang.org/x/sys/windows"
)

func bytePtrFromString(s string) (*byte, error) {
	if s == "" {
		return new(byte), nil
	}

	ptr, err := windows.BytePtrFromString(s)
	if err != nil {
		return nil, err
	}

	return ptr, nil
}

func bytePtrToString(p *byte) string {
	if p == nil {
		return ""
	}
	return windows.BytePtrToString(p)
}
