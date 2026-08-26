//go:build windows

package platform

import "syscall"

func DetachAttr() *syscall.SysProcAttr {
	const (
		createNewProcessGroup = 0x00000200
		createNoWindow        = 0x08000000
	)
	return &syscall.SysProcAttr{
		CreationFlags: createNewProcessGroup | createNoWindow,
		HideWindow:    true,
	}
}
