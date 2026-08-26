package proc

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func SelfPID() int { return os.Getpid() }

func PortFromURL(raw string) int {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return 0
	}
	host := u.Host
	if strings.HasPrefix(host, "[") {
		if i := strings.LastIndex(host, "]:"); i >= 0 {
			p, _ := strconv.Atoi(host[i+2:])
			return p
		}
		return 0
	}
	_, port, err := net.SplitHostPort(host)
	if err != nil {
		if u.Scheme == "https" {
			return 443
		}
		if u.Scheme == "http" {
			return 80
		}
		return 0
	}
	p, _ := strconv.Atoi(port)
	return p
}

func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		return windowsAlive(pid)
	}
	return unixAlive(pid)
}

func PIDsOnPort(port int) []int {
	if port <= 0 {
		return nil
	}
	if runtime.GOOS == "windows" {
		return pidsOnPortWindows(port)
	}
	return pidsOnPortUnix(port)
}

func KillPID(pid int) error {
	if pid <= 0 || pid == SelfPID() {
		return nil
	}
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
	}
	return unixKill(pid)
}

func KillAll(pids []int) []int {
	seen := map[int]struct{}{}
	var killed []int
	for _, pid := range pids {
		if pid <= 0 || pid == SelfPID() {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		if KillPID(pid) == nil || !Alive(pid) {
			killed = append(killed, pid)
		}
	}
	return killed
}

func FindByNeedle(needle string) []int {
	needle = strings.TrimSpace(needle)
	if needle == "" || len(needle) < 4 {
		return nil
	}
	if runtime.GOOS == "windows" {
		return findWindows(needle)
	}
	return findUnix(needle)
}

func DistinctiveNeedle(argv []string) string {
	for i := len(argv) - 1; i >= 0; i-- {
		base := filepath.Base(argv[i])
		if strings.Contains(base, ".") && i > 0 {
			return base
		}
	}
	if len(argv) > 0 {
		base := filepath.Base(argv[0])
		switch strings.ToLower(base) {
		case "python", "python3", "pythonw", "node", "bash", "sh", "zsh", "cmd", "cmd.exe", "powershell":
			return ""
		default:
			if len(base) >= 4 {
				return base
			}
		}
	}
	return ""
}

func pidsOnPortUnix(port int) []int {
	out, err := exec.Command("lsof", "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-t").Output()
	if err != nil {
		return nil
	}
	return parsePIDLines(string(out))
}

func pidsOnPortWindows(port int) []int {
	out, err := exec.Command("netstat", "-ano", "-p", "tcp").Output()
	if err != nil {
		return nil
	}
	return ParseNetstat(string(out), port)
}

func ParseNetstat(out string, port int) []int {
	want := ":" + strconv.Itoa(port)
	seen := map[int]struct{}{}
	var pids []int
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.Contains(strings.ToUpper(line), "LISTEN") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		local := fields[1]
		if len(fields) >= 2 && strings.EqualFold(fields[0], "TCP") {
			local = fields[1]
		}
		if !portMatches(local, want) {
			continue
		}
		pid, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil || pid <= 0 {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		pids = append(pids, pid)
	}
	return pids
}

func portMatches(local, want string) bool {
	if strings.HasSuffix(local, want) {
		// avoid 18765 matching :8765 — suffix after last ':'
		i := strings.LastIndex(local, ":")
		if i < 0 {
			return false
		}
		return ":"+local[i+1:] == want
	}
	return false
}

func parsePIDLines(out string) []int {
	seen := map[int]struct{}{}
	var pids []int
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil || pid <= 0 {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		pids = append(pids, pid)
	}
	return pids
}

func findUnix(needle string) []int {
	out, err := exec.Command("ps", "-ax", "-o", "pid=,command=").Output()
	if err != nil {
		return nil
	}
	return parsePS(string(out), needle)
}

func ParsePS(out, needle string) []int {
	return parsePS(out, needle)
}

func parsePS(out, needle string) []int {
	self := strconv.Itoa(SelfPID())
	var pids []int
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !strings.Contains(line, needle) {
			continue
		}
		pidStr, rest, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		if pidStr == self {
			continue
		}
		if strings.Contains(rest, "devtoolbox") && !strings.Contains(needle, "devtoolbox") {
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}

func findWindows(needle string) []int {
	q := strings.ReplaceAll(needle, "'", "''")
	ps := fmt.Sprintf(`Get-CimInstance Win32_Process | Where-Object { $_.CommandLine -like '*%s*' } | Select-Object -ExpandProperty ProcessId`, q)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", ps).Output()
	if err != nil {
		return nil
	}
	return parsePIDLines(string(out))
}

func windowsAlive(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false
	}
	s := string(out)
	return strings.Contains(s, strconv.Itoa(pid)) && !strings.Contains(strings.ToLower(s), "no tasks")
}

func QuitApp(appPath string) {
	appPath = strings.TrimSpace(appPath)
	if appPath == "" {
		return
	}
	base := filepath.Base(appPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("osascript", "-e", `tell application `+strconv.Quote(name)+` to quit`).Run()
	case "windows":
		_ = exec.Command("taskkill", "/IM", base, "/T", "/F").Run()
	}
}
