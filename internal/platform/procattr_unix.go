//go:build !windows

package platform

import "syscall"

func DetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
