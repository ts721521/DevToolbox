package launcher

import (
	"fmt"
	"strings"
	"time"

	"github.com/ts721521/DevToolbox/internal/applog"
	"github.com/ts721521/DevToolbox/internal/platform"
	"github.com/ts721521/DevToolbox/internal/proc"
	"github.com/ts721521/DevToolbox/internal/registry"
)

func startServices(t registry.Tool) error {
	if len(t.Services) == 0 {
		return nil
	}
	var pids []int
	for i, s := range t.Services {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			name = fmt.Sprintf("svc-%d", i+1)
		}
		if serviceRunning(s) {
			applog.Info("service_already", "id", t.ID, "service", name)
			continue
		}
		if len(s.Command) == 0 {
			return fmt.Errorf("附加服务「%s」没有 command", name)
		}
		applog.Info("service_launch", "id", t.ID, "service", name, "cmd", strings.Join(s.Command, " "))
		var pid int
		var err error
		if s.Terminal {
			err = platform.RunInTerminal(s.Workdir, s.Command, s.Env)
		} else {
			pid, err = platform.StartDetached(s.Workdir, s.Command, s.Env)
		}
		if err != nil {
			rollbackServices(t, pids)
			return fmt.Errorf("启动附加服务「%s」失败：%w", name, err)
		}
		if pid > 0 {
			pids = append(pids, pid)
		}
		if err := waitService(s); err != nil {
			rollbackServices(t, pids)
			return err
		}
		if n := serviceNeedle(s); n != "" {
			pids = append(pids, proc.FindByNeedle(n)...)
		}
		port := proc.LocalListenPort(s.HealthURL)
		if port > 0 && port != 17890 {
			pids = append(pids, proc.PIDsOnPort(port)...)
		}
	}
	if len(pids) > 0 {
		rec := loadRun(t.ID)
		rec.ID = t.ID
		rec.ServicePIDs = uniqueInts(append(rec.ServicePIDs, pids...))
		if rec.StartedAt.IsZero() {
			rec.StartedAt = time.Now()
		}
		saveRun(rec)
	}
	return nil
}

func stopServices(t registry.Tool) {
	if len(t.Services) == 0 {
		rec := loadRun(t.ID)
		if len(rec.ServicePIDs) > 0 {
			_ = proc.KillAll(rec.ServicePIDs)
		}
		return
	}
	rec := loadRun(t.ID)
	pids := append([]int{}, rec.ServicePIDs...)
	for _, s := range t.Services {
		if n := serviceNeedle(s); n != "" {
			pids = append(pids, proc.FindByNeedle(n)...)
		}
		port := proc.LocalListenPort(s.HealthURL)
		if port > 0 && port != 17890 {
			include := s.HealthContains == ""
			if s.HealthURL != "" {
				ok, _ := httpOK(s.HealthURL, s.HealthContains)
				include = ok || s.HealthContains == ""
			}
			if include {
				pids = append(pids, proc.PIDsOnPort(port)...)
			}
		}
	}
	killed := proc.KillAll(uniqueInts(pids))
	applog.Info("services_stop", "id", t.ID, "killed", killed)
}

func rollbackServices(t registry.Tool, pids []int) {
	if len(pids) > 0 {
		rec := loadRun(t.ID)
		rec.ID = t.ID
		rec.ServicePIDs = uniqueInts(append(rec.ServicePIDs, pids...))
		saveRun(rec)
	}
	stopServices(t)
	rec := loadRun(t.ID)
	rec.ID = t.ID
	rec.ServicePIDs = nil
	if len(rec.PIDs) == 0 {
		clearRun(t.ID)
		return
	}
	saveRun(rec)
}

func serviceRunning(s registry.Service) bool {
	if strings.TrimSpace(s.HealthURL) != "" {
		ok, _ := httpOK(s.HealthURL, s.HealthContains)
		if ok {
			return true
		}
	}
	if n := serviceNeedle(s); n != "" && len(proc.FindByNeedle(n)) > 0 {
		return true
	}
	return false
}

func waitService(s registry.Service) error {
	if strings.TrimSpace(s.HealthURL) != "" {
		d := 25 * time.Second
		if s.WaitMS > 0 {
			d = time.Duration(s.WaitMS) * time.Millisecond
		}
		if d > 60*time.Second {
			d = 60 * time.Second
		}
		deadline := time.Now().Add(d)
		for time.Now().Before(deadline) {
			if ok, _ := httpOK(s.HealthURL, s.HealthContains); ok {
				return nil
			}
			time.Sleep(400 * time.Millisecond)
		}
		name := strings.TrimSpace(s.Name)
		if name == "" {
			name = s.HealthURL
		}
		return fmt.Errorf("附加服务「%s」健康检查超时", name)
	}
	ms := s.WaitMS
	if ms <= 0 {
		ms = 800
	}
	if ms > 60_000 {
		ms = 60_000
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return nil
}

func serviceNeedle(s registry.Service) string {
	if strings.TrimSpace(s.ProcessMatch) != "" {
		return s.ProcessMatch
	}
	return proc.DistinctiveNeedle(s.Command)
}
