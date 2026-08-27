package registry

import (
	"errors"
	"testing"
)

func TestRemoveBlocksReregister(t *testing.T) {
	withTempConfig(t)
	wd := t.TempDir()
	tool := Tool{ID: "gone", Name: "Gone", Kind: "url", URL: "http://127.0.0.1:9", Workdir: wd, Git: "https://github.com/org/gone.git"}
	if err := Save(tool); err != nil {
		t.Fatal(err)
	}
	if err := Remove("gone"); err != nil {
		t.Fatal(err)
	}
	if _, err := Get("gone"); err == nil {
		t.Fatal("expected not found")
	}
	if err := Save(tool); err == nil || !errors.Is(err, ErrBlocked) {
		t.Fatalf("same id should be blocked: %v", err)
	}
	alias := Tool{ID: "gone-2", Name: "Gone2", Kind: "url", URL: "http://127.0.0.1:9", Workdir: wd}
	if err := Save(alias); err == nil || !errors.Is(err, ErrBlocked) {
		t.Fatalf("same workdir should be blocked: %v", err)
	}
	gitTwin := Tool{ID: "gone-3", Name: "Gone3", Kind: "url", URL: "http://127.0.0.1:9", Git: "git@github.com:org/gone.git"}
	if err := Save(gitTwin); err == nil || !errors.Is(err, ErrBlocked) {
		t.Fatalf("same git should be blocked: %v", err)
	}
}

func TestRestoreFromTrash(t *testing.T) {
	withTempConfig(t)
	wd := t.TempDir()
	tool := Tool{ID: "back", Name: "Back", Kind: "url", URL: "http://127.0.0.1:9", Workdir: wd}
	if err := Save(tool); err != nil {
		t.Fatal(err)
	}
	if err := Remove("back"); err != nil {
		t.Fatal(err)
	}
	if err := Restore("back"); err != nil {
		t.Fatal(err)
	}
	got, err := Get("back")
	if err != nil || got.Name != "Back" {
		t.Fatalf("got %+v err=%v", got, err)
	}
	if err := Save(tool); err != nil {
		t.Fatal(err)
	}
}

func TestUnblockAllowsRegister(t *testing.T) {
	withTempConfig(t)
	tool := Tool{ID: "skip", Name: "Skip", Kind: "url", URL: "http://127.0.0.1:9"}
	if err := Save(tool); err != nil {
		t.Fatal(err)
	}
	if err := Remove("skip"); err != nil {
		t.Fatal(err)
	}
	if err := Unblock("skip"); err != nil {
		t.Fatal(err)
	}
	if err := Save(tool); err != nil {
		t.Fatal(err)
	}
}

func TestSiblingSameWorkdirStillSavable(t *testing.T) {
	withTempConfig(t)
	wd := t.TempDir()
	a := Tool{ID: "keep", Name: "Keep", Kind: "url", URL: "http://127.0.0.1:11", Workdir: wd}
	b := Tool{ID: "drop", Name: "Drop", Kind: "url", URL: "http://127.0.0.1:12", Workdir: wd}
	if err := Save(a); err != nil {
		t.Fatal(err)
	}
	if err := Save(b); err != nil {
		t.Fatal(err)
	}
	if err := Remove("drop"); err != nil {
		t.Fatal(err)
	}
	a.Name = "Keep2"
	if err := Save(a); err != nil {
		t.Fatalf("live sibling must still save: %v", err)
	}
}

func TestBlockedSameURL(t *testing.T) {
	withTempConfig(t)
	tool := Tool{ID: "u1", Name: "U1", Kind: "url", URL: "http://127.0.0.1:8765/"}
	if err := Save(tool); err != nil {
		t.Fatal(err)
	}
	if err := Remove("u1"); err != nil {
		t.Fatal(err)
	}
	again := Tool{ID: "u2", Name: "U2", Kind: "url", URL: "http://127.0.0.1:8765"}
	if err := Save(again); err == nil || !errors.Is(err, ErrBlocked) {
		t.Fatalf("same url should be blocked: %v", err)
	}
}

func TestTrashRejectsTraversal(t *testing.T) {
	withTempConfig(t)
	if _, err := trashPath("../etc"); err == nil {
		t.Fatal("expected invalid")
	}
	if err := Restore("../etc"); err == nil {
		t.Fatal("expected invalid")
	}
	if err := Unblock("../etc"); err == nil {
		t.Fatal("expected invalid")
	}
}
