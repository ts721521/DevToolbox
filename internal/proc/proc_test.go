package proc

import "testing"

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
