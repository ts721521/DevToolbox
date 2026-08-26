package registry

import (
	"testing"
)

func TestTabsMoveAndRename(t *testing.T) {
	withTempConfig(t)
	if err := Save(Tool{ID: "a", Name: "A", Kind: "url", URL: "http://127.0.0.1:1", Group: "工作"}); err != nil {
		t.Fatal(err)
	}
	if err := Save(Tool{ID: "b", Name: "B", Kind: "url", URL: "http://127.0.0.1:2", Group: "财务"}); err != nil {
		t.Fatal(err)
	}
	tabs := LoadTabs()
	if !containsTab(tabs, "工作") || !containsTab(tabs, "财务") || !containsTab(tabs, DefaultOther) {
		t.Fatalf("tabs=%v", tabs)
	}
	if err := MoveTools([]string{"a"}, "开发"); err != nil {
		t.Fatal(err)
	}
	got, _ := Get("a")
	if got.Group != "开发" {
		t.Fatalf("group=%s", got.Group)
	}
	if err := RenameTab("开发", "工具"); err != nil {
		t.Fatal(err)
	}
	got, _ = Get("a")
	if got.Group != "工具" {
		t.Fatalf("renamed=%s", got.Group)
	}
	if err := RemoveTab("工具", "财务"); err != nil {
		t.Fatal(err)
	}
	got, _ = Get("a")
	if got.Group != "财务" {
		t.Fatalf("moved=%s", got.Group)
	}
	if containsTab(LoadTabs(), "工具") {
		t.Fatal("deleted tab still present")
	}
}

func TestRemoveTabPersistsNewMoveTo(t *testing.T) {
	withTempConfig(t)
	if err := Save(Tool{ID: "a", Name: "A", Kind: "url", URL: "http://127.0.0.1:1", Group: "工作"}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveTab("工作", "临时仓"); err != nil {
		t.Fatal(err)
	}
	got, _ := Get("a")
	if got.Group != "临时仓" {
		t.Fatalf("group=%s", got.Group)
	}
	saved := readTabFile()
	if !containsTab(saved, "临时仓") {
		t.Fatalf("tabs.json missing 临时仓: %v", saved)
	}
}

func TestRenameTabSameNameKeepsTab(t *testing.T) {
	withTempConfig(t)
	if _, err := AddTab("空标签"); err != nil {
		t.Fatal(err)
	}
	if err := RenameTab("空标签", "空标签"); err != nil {
		t.Fatal(err)
	}
	if !containsTab(LoadTabs(), "空标签") {
		t.Fatalf("lost tab: %v", LoadTabs())
	}
}

func TestMoveToolsRejectsBlankIDs(t *testing.T) {
	withTempConfig(t)
	if err := MoveTools([]string{"", " "}, "开发"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCannotDeleteOther(t *testing.T) {
	withTempConfig(t)
	if err := RemoveTab(DefaultOther, "工作"); err == nil {
		t.Fatal("expected error")
	}
}

func containsTab(tabs []string, name string) bool {
	for _, n := range tabs {
		if n == name {
			return true
		}
	}
	return false
}
