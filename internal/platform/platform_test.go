package platform

import (
	"testing"
)

func TestHubLaunchedByCocoa(t *testing.T) {
	if !HubLaunchedByCocoa("/Applications/工坞.app/Contents/Helpers/tooldock") {
		t.Fatal("helpers path is cocoa hub")
	}
	if HubLaunchedByCocoa("/usr/local/bin/tooldock") {
		t.Fatal("cli must still open the UI")
	}
}
