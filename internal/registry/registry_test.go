package registry

import (
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

func TestValidateRejectsBadID(t *testing.T) {
	err := Validate(Tool{ID: "../etc", Name: "x", Kind: "url", URL: "http://x"})
	if err == nil {
		t.Fatal("expected error")
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
