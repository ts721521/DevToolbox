package launcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ts721521/DevToolbox/internal/registry"
)

func TestDirToRevealUsesWorkdir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	got, err := DirToReveal(registry.Tool{Workdir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("got %s", got)
	}
}

func TestDirToRevealMissing(t *testing.T) {
	if _, err := DirToReveal(registry.Tool{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestProgramToReveal(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "app.bin")
	if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ProgramToReveal(registry.Tool{AppPath: p})
	if err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Fatalf("got %s", got)
	}
	if _, err := ProgramToReveal(registry.Tool{}); err == nil {
		t.Fatal("expected error")
	}
}
