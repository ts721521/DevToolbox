package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	// Windows uses APPDATA; macOS UserConfigDir uses HOME/Library/Application Support
	t.Setenv("APPDATA", dir)
	t.Setenv("USERPROFILE", dir)
}

func TestSaveListGetRemove(t *testing.T) {
	withTempConfig(t)
	wd := t.TempDir()

	tool := Tool{
		ID:      "highspot-sync",
		Name:    "AVEVA Highspot 同步",
		Kind:    "web",
		URL:     "http://localhost:8765",
		Workdir: wd,
	}
	if err := Save(tool); err != nil {
		t.Fatal(err)
	}
	got, err := Get("highspot-sync")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != tool.Name || got.URL != tool.URL {
		t.Fatalf("got %+v", got)
	}
	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("len=%d", len(list))
	}
	if err := Remove("highspot-sync"); err != nil {
		t.Fatal(err)
	}
	if _, err := Get("highspot-sync"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestPathForRejectsTraversal(t *testing.T) {
	withTempConfig(t)
	if _, err := Get("../etc"); err == nil {
		t.Fatal("expected invalid id")
	}
	if err := Remove("../etc"); err == nil {
		t.Fatal("expected invalid id")
	}
}

func TestValidateRejectsJavascriptURL(t *testing.T) {
	err := Validate(Tool{ID: "x", Name: "X", Kind: "url", URL: "javascript:alert(1)"})
	if err == nil {
		t.Fatal("expected javascript url rejected")
	}
	err = Validate(Tool{ID: "x", Name: "X", Kind: "url", URL: "file:///etc/passwd"})
	if err == nil {
		t.Fatal("expected file url rejected")
	}
}

func TestSaveScrubGitPersists(t *testing.T) {
	withTempConfig(t)
	if err := Save(Tool{
		ID:   "g",
		Name: "G",
		Kind: "url",
		URL:  "http://127.0.0.1:9",
		Git:  "https://user:token@github.com/org/repo.git",
	}); err != nil {
		t.Fatal(err)
	}
	p, err := pathFor("g")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "user:token") || strings.Contains(string(raw), "token@") {
		t.Fatalf("credentials persisted: %s", raw)
	}
	got, err := Get("g")
	if err != nil {
		t.Fatal(err)
	}
	if got.Git != "https://github.com/org/repo.git" {
		t.Fatalf("git=%s", got.Git)
	}
}

func TestHealthURLMustBeLocal(t *testing.T) {
	err := Validate(Tool{
		ID: "x", Name: "X", Kind: "url", URL: "https://example.com",
		HealthURL: "https://example.com/health",
	})
	if err == nil {
		t.Fatal("remote health_url should fail")
	}
}

func TestValidateServiceHealthURLMustBeLocal(t *testing.T) {
	err := Validate(Tool{
		ID: "x", Name: "X", Kind: "url", URL: "http://127.0.0.1:9",
		Services: []Service{{
			Command:   []string{"sleep", "1"},
			HealthURL: "https://evil.example/health",
		}},
	})
	if err == nil {
		t.Fatal("remote services.health_url should fail")
	}
}

func TestValidateCommandDarwinOnly(t *testing.T) {
	wd := t.TempDir()
	err := Validate(Tool{
		ID: "x", Name: "X", Kind: "command", Workdir: wd,
		CommandDarwin: []string{"echo", "ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRejectsInvalidDiskJSON(t *testing.T) {
	withTempConfig(t)
	dir, err := EnsureDir()
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "bad.json")
	body := `{"id":"bad","name":"Bad","kind":"url","url":"javascript:alert(1)"}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Get("bad"); err == nil {
		t.Fatal("expected invalid")
	}
}

func TestValidateRejectsBadID(t *testing.T) {
	err := Validate(Tool{ID: "../etc", Name: "x", Kind: "url", URL: "http://x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtrasHint(t *testing.T) {
	empty := Tool{ID: "x", Name: "X", Kind: "url", URL: "http://127.0.0.1:9"}
	if ExtrasHint(empty) == "" {
		t.Fatal("expected hint when no extras")
	}
	withSvc := empty
	withSvc.Services = []Service{{Command: []string{"sleep", "1"}}}
	if ExtrasHint(withSvc) != "" {
		t.Fatal("no hint when services present")
	}
	withDep := empty
	withDep.Depends = []string{"other"}
	if ExtrasHint(withDep) != "" {
		t.Fatal("no hint when depends present")
	}
}

func TestValidateRequiresWorkdir(t *testing.T) {
	err := Validate(Tool{ID: "x", Name: "X", Kind: "web", URL: "http://127.0.0.1:1"})
	if err == nil {
		t.Fatal("expected workdir required")
	}
	err = Validate(Tool{ID: "y", Name: "Y", Kind: "url", URL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateServicesAndDepends(t *testing.T) {
	ok := Tool{
		ID:      "app",
		Name:    "App",
		Kind:    "url",
		URL:     "http://127.0.0.1:1",
		Depends: []string{"redis"},
		Services: []Service{{
			Name:    "worker",
			Command: []string{"python3", "worker.py"},
		}},
	}
	if err := Validate(ok); err != nil {
		t.Fatal(err)
	}
	if err := Validate(Tool{ID: "a", Name: "A", Kind: "url", URL: "http://x", Depends: []string{"a"}}); err == nil {
		t.Fatal("expected self-depend error")
	}
	if err := Validate(Tool{ID: "a", Name: "A", Kind: "url", URL: "http://x", Services: []Service{{}}}); err == nil {
		t.Fatal("expected empty service command error")
	}
}

func TestInferKind(t *testing.T) {
	tool := Tool{ID: "x", Name: "X", URL: "http://localhost:1", Command: []string{"python3", "s.py"}}
	tool = Normalize(tool)
	if tool.Kind != "web" {
		t.Fatalf("kind=%s", tool.Kind)
	}
	if !tool.Supports("darwin") || !tool.Supports("windows") {
		t.Fatal("default platforms should include macOS and Windows")
	}
}

func TestPlatformAliases(t *testing.T) {
	got := NormalizePlatforms([]string{"macOS", "Win"})
	if len(got) != 2 || got[0] != "darwin" || got[1] != "windows" {
		t.Fatalf("%v", got)
	}
	t2 := Tool{ID: "a", Name: "A", Kind: "url", URL: "http://x", Platforms: []string{"windows"}}
	t2 = Normalize(t2)
	if t2.Supports("darwin") || !t2.Supports("windows") {
		t.Fatal("windows-only")
	}
}
