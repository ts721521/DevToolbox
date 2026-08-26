package applog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathAndTail(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("USERPROFILE", dir)
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	Info("hello", "k", 1)
	Error("boom", "err", "x")
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p, "tooldock.log") {
		t.Fatalf("path=%s", p)
	}
	text, err := Tail(50)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "hello") || !strings.Contains(text, "boom") {
		t.Fatalf("tail=%q", text)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(p), "tooldock.log")); err != nil {
		t.Fatal(err)
	}
}
