//go:build !windows

package proc

import (
	"os"
	"syscall"
	"time"
)

func unixAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func unixKill(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	_ = p.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	return p.Signal(syscall.SIGKILL)
}
