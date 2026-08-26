package server

import (
	"strings"
	"testing"
)

func TestAIPromptContainsLocalURL(t *testing.T) {
	p := AIPrompt()
	for _, part := range []string{ForAIURL(), "tooldock register", ".devtoolbox.json"} {
		if !strings.Contains(p, part) {
			t.Fatalf("missing %q in %s", part, p)
		}
	}
}

func TestForAIMarkdownHasAPI(t *testing.T) {
	md := forAIMarkdown()
	for _, part := range []string{"/api/tools", "工坞", "/api/logs", "tooldock.log", "/api/tabs", "group", "/dir", "workdir", "app_path", "tooldock dir", "tooldock app"} {
		if !strings.Contains(md, part) {
			t.Fatalf("markdown missing %q", part)
		}
	}
}
