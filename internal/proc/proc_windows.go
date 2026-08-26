//go:build windows

package proc

func unixAlive(pid int) bool { return windowsAlive(pid) }

func unixKill(pid int) error {
	return KillPID(pid)
}
