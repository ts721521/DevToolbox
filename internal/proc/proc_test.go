package proc

import "testing"

func TestLocalListenPort(t *testing.T) {
	if LocalListenPort("http://localhost:8765/") != 8765 {
		t.Fatal("8765")
	}
	if LocalListenPort("http://127.0.0.1:17890") != 0 {
		t.Fatal("hub port must not be killable")
	}
	if LocalListenPort("http://localhost") != 0 {
		t.Fatal("no inferred 80")
	}
	if LocalListenPort("https://github.com") != 0 {
		t.Fatal("no inferred 443")
	}
	if LocalListenPort("https://example.com:443") != 0 {
		t.Fatal("remote 443 must not be killable")
	}
}

func TestPortFromURL(t *testing.T) {
	if PortFromURL("http://localhost:8765/") != 8765 {
		t.Fatal("8765")
	}
	if PortFromURL("http://127.0.0.1:17890") != 17890 {
		t.Fatal("17890")
	}
	if PortFromURL("http://localhost") != 80 {
		t.Fatal("80")
	}
}

func TestParseNetstatDoesNotMatchPartialPort(t *testing.T) {
	out := `
  TCP    127.0.0.1:18765        0.0.0.0:0              LISTENING       11
  TCP    127.0.0.1:8765         0.0.0.0:0              LISTENING       22
  TCP    [::]:8765              [::]:0                 LISTENING       33
`
	pids := ParseNetstat(out, 8765)
	if len(pids) != 2 {
		t.Fatalf("got %v", pids)
	}
}

func TestDistinctiveNeedle(t *testing.T) {
	if DistinctiveNeedle([]string{"/usr/bin/python3", "server.py"}) != "server.py" {
		t.Fatal("server.py")
	}
	if DistinctiveNeedle([]string{"python3"}) != "" {
		t.Fatal("bare python should be empty")
	}
}

func TestParsePSSkipsHubProcess(t *testing.T) {
	out := "100 /Applications/工坞.app/Contents/MacOS/tooldock\n200 python3 myserver.py\n300 /usr/local/bin/devtoolbox\n"
	got := ParsePS(out, "python3")
	if len(got) != 1 || got[0] != 200 {
		t.Fatalf("got %v", got)
	}
	got = ParsePS(out, "tooldock")
	if len(got) != 1 || got[0] != 100 {
		t.Fatalf("explicit tooldock needle: %v", got)
	}
}
