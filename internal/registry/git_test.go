package registry

import "testing"

func TestGitWebURL(t *testing.T) {
	got := GitWebURL("git@github.com:ts721521/DevToolbox.git")
	if got != "https://github.com/ts721521/DevToolbox" {
		t.Fatalf("got %s", got)
	}
	if GitLabel("https://github.com/ts721521/DevToolbox.git") != "ts721521/DevToolbox" {
		t.Fatalf("label=%s", GitLabel("https://github.com/ts721521/DevToolbox.git"))
	}
	if GitWebURL("GIT@github.com:ts721521/DevToolbox.git") != "https://github.com/ts721521/DevToolbox" {
		t.Fatal("git@ case")
	}
	if ScrubGit("ssh://user:pass@host/org/repo.git") != "ssh://host/org/repo.git" {
		t.Fatalf("ssh scrub=%s", ScrubGit("ssh://user:pass@host/org/repo.git"))
	}
	if err := Validate(Tool{ID: "a", Name: "A", Kind: "url", URL: "http://x", Git: "javascript:x"}); err == nil {
		t.Fatal("expected git error")
	}
}

func TestScrubGitStripsUserinfo(t *testing.T) {
	in := "https://user:token@github.com/org/repo.git"
	if ScrubGit(in) != "https://github.com/org/repo.git" {
		t.Fatalf("got %s", ScrubGit(in))
	}
	if GitWebURL(in) != "https://github.com/org/repo" {
		t.Fatalf("web=%s", GitWebURL(in))
	}
}
