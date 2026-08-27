package version

import "testing"

func TestDisplayAndNumeric(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })

	Version = "1.2.0"
	if Display() != "v1.2.0" || Numeric() != "1.2.0" || Tag() != "v1.2.0" {
		t.Fatalf("display=%s numeric=%s", Display(), Numeric())
	}
	Version = "v1.2.0"
	if Display() != "v1.2.0" || Numeric() != "1.2.0" {
		t.Fatalf("display=%s numeric=%s", Display(), Numeric())
	}
	Version = "v1.2.0-dev"
	if Numeric() != "1.2.0" {
		t.Fatalf("numeric=%s", Numeric())
	}
}

func TestDisplayEmptyUnknown(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })
	Version = ""
	if Display() != "v0.0.0-dev" {
		t.Fatalf("empty → %s", Display())
	}
	Version = "unknown"
	if Display() != "v0.0.0-dev" {
		t.Fatalf("unknown → %s", Display())
	}
}

func TestArtifactNames(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })
	Version = "1.2.0"
	if BinaryName("darwin", "arm64") != "tooldock-darwin-arm64-v1.2.0" {
		t.Fatalf("%s", BinaryName("darwin", "arm64"))
	}
	if BinaryName("windows", "amd64") != "tooldock-windows-amd64-v1.2.0.exe" {
		t.Fatalf("%s", BinaryName("windows", "amd64"))
	}
	if AppBundleName() != "工坞-v1.2.0.app" {
		t.Fatalf("%s", AppBundleName())
	}
	if ZipName("darwin", "arm64") != "ToolDock-v1.2.0-macOS-arm64.zip" {
		t.Fatalf("%s", ZipName("darwin", "arm64"))
	}
}
