package proc

import (
	"bufio"
	"fmt"
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
	if p := u.Port(); p != "" {
		n, _ := strconv.Atoi(p)
		return n
	}
	if u.Scheme == "https" {
		return 443
	}
	if u.Scheme == "http" {
		return 80
	}
	return 0
}

// LocalListenPort is the only port Stop/Probe may kill or claim.
// Requires an explicit port on loopback. Never infers 80/443.
func LocalListenPort(raw string) int {
	u, err := url.Parse(raw)
	if err != nil {
		return 0
	}
	host := strings.ToLower(u.Hostname())
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return 0
	}
	if u.Port() == "" {
		return 0
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil || p <= 0 || p == 17890 {
		return 0
	}
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

func skipHubLine(cmdline, needle string) bool {
	cl := strings.ToLower(cmdline)
	n := strings.ToLower(needle)
	for _, hub := range []string{"tooldock", "devtoolbox"} {
		if strings.Contains(cl, hub) && !strings.Contains(n, hub) {
			return true
		}
	}
	return false
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
		if skipHubLine(rest, needle) {
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
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return nil
	}
	// Filter in Go so the needle never enters a PowerShell string.
	ps := `Get-CimInstance Win32_Process | ForEach-Object { '{0}\t{1}' -f $_.ProcessId, $_.CommandLine }`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", ps).Output()
	if err != nil {
		return nil
	}
	var pids []int
	self := strconv.Itoa(SelfPID())
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		pidStr, rest, ok := strings.Cut(line, "\t")
		if !ok || pidStr == self {
			continue
		}
		if !strings.Contains(rest, needle) {
			continue
		}
		if skipHubLine(rest, needle) {
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
