package server

import (
	"strings"
	"testing"
)

func TestAIPromptContainsLocalURL(t *testing.T) {
	p := AIPrompt()
	for _, part := range []string{ForAIURL(), "tooldock register", ".devtoolbox.json", "Origin: http://127.0.0.1:17890", "/api/tools", "services", "/api/blocked"} {
		if !strings.Contains(p, part) {
			t.Fatalf("missing %q in %s", part, p)
		}
	}
}

func TestForAIMarkdownHasAPI(t *testing.T) {
	md := forAIMarkdown()
	for _, part := range []string{"/api/tools", "工坞", "/api/logs", "tooldock.log", "/api/tabs", "group", "/dir", "workdir", "app_path", "tooldock dir", "tooldock app", "services", "depends", "git", "tooldock git", "Origin: http://127.0.0.1:17890", "X-ToolDock-Token", "不会自动出现", "禁止编造", "/api/blocked", "垃圾桶"} {
		if !strings.Contains(md, part) {
			t.Fatalf("markdown missing %q", part)
		}
	}
}
