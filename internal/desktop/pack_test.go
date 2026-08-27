package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ts721521/DevToolbox/internal/version"
)

func TestPackBinaries(t *testing.T) {
	src := filepath.Join(t.TempDir(), "tooldock-src")
	if err := os.WriteFile(src, []byte("fake-bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	written, err := packBinaries(out, src)
	if err != nil {
		t.Fatal(err)
	}
	want := version.BinaryName(runtime.GOOS, runtime.GOARCH)
	found := false
	for _, p := range written {
		if filepath.Base(p) == want {
			found = true
			got, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "fake-bin" {
				t.Fatalf("copied bytes = %q", got)
			}
		}
	}
	if !found {
		t.Fatalf("missing %s in %v", want, written)
	}
}

func TestWriteChecksums(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := writeChecksums(dir, []string{p, dir})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "a.bin") {
		t.Fatalf("%s", raw)
	}
}

func TestDarwinBundleHasCocoaLauncher(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bin")
	if err := os.WriteFile(src, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(dir, "工坞.app")
	if err := writeDarwinBundle(app, src); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(app, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "LSUIElement") {
		t.Fatal("Dock icon requires LSUIElement off")
	}
	if runtime.GOOS == "darwin" && !strings.Contains(string(raw), "NSAllowsLocalNetworking") {
		t.Fatal("WKWebView needs NSAllowsLocalNetworking for http://127.0.0.1")
	}
	if _, err := os.Stat(filepath.Join(app, "Contents", "Helpers", BinName)); err != nil {
		t.Fatal("missing go helper binary")
	}
}
