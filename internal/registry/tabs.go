package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultOther = "其他"
	tabsFile     = "tabs.json"
)

// DefaultTabs is the built-in set. Keep it short; AIs must reuse these
// names instead of inventing new ones.
var DefaultTabs = []string{"工作", "财务", "开发", DefaultOther}

type tabStore struct {
	Tabs []string `json:"tabs"`
}

func tabsPath() (string, error) {
	root, err := RootDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(root, tabsFile), nil
}

func cleanTabName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "\\", "")
	if len([]rune(s)) > 16 {
		s = string([]rune(s)[:16])
	}
	return s
}

func uniqueTabs(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, n := range in {
		n = cleanTabName(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func readTabFile() []string {
	p, err := tabsPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var st tabStore
	if json.Unmarshal(data, &st) != nil {
		return nil
	}
	return uniqueTabs(st.Tabs)
}

func SaveTabs(names []string) error {
	names = uniqueTabs(names)
	if len(names) == 0 {
		names = append([]string{}, DefaultTabs...)
	}
	hasOther := false
	for _, n := range names {
		if n == DefaultOther {
			hasOther = true
			break
		}
	}
	if !hasOther {
		names = append(names, DefaultOther)
	}
	p, err := tabsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(tabStore{Tabs: names}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}

// LoadTabs returns the label bar: saved order, then any extra groups from tools.
func LoadTabs() []string {
	saved := readTabFile()
	if len(saved) == 0 {
		saved = append([]string{}, DefaultTabs...)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(saved)+4)
	add := func(n string) {
		n = cleanTabName(n)
		if n == "" {
			return
		}
		if _, ok := seen[n]; ok {
			return
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	for _, n := range saved {
		add(n)
	}
	tools, err := List()
	if err == nil {
		for _, t := range tools {
			add(t.Group)
		}
	}
	add(DefaultOther)
	return out
}

func TabName(s string) string {
	return cleanTabName(s)
}

func withTab(names []string, extra string) []string {
	extra = cleanTabName(extra)
	if extra == "" {
		return names
	}
	for _, n := range names {
		if n == extra {
			return names
		}
	}
	return append(names, extra)
}

func EnsureTab(name string) error {
	name = cleanTabName(name)
	if name == "" {
		return nil
	}
	cur := LoadTabs()
	for _, n := range cur {
		if n == name {
			return nil
		}
	}
	return SaveTabs(append(cur, name))
}

func AddTab(name string) (string, error) {
	name = cleanTabName(name)
	if name == "" {
		return "", fmt.Errorf("%w: tab name required", ErrInvalid)
	}
	cur := LoadTabs()
	for _, n := range cur {
		if n == name {
			return name, nil
		}
	}
	if err := SaveTabs(append(cur, name)); err != nil {
		return "", err
	}
	return name, nil
}

func RenameTab(from, to string) error {
	from = cleanTabName(from)
	to = cleanTabName(to)
	if from == "" || to == "" {
		return fmt.Errorf("%w: tab name required", ErrInvalid)
	}
	if from == DefaultOther {
		return fmt.Errorf("%w: cannot rename %s", ErrInvalid, DefaultOther)
	}
	if from == to {
		return nil
	}
	cur := LoadTabs()
	found := false
	next := make([]string, 0, len(cur))
	for _, n := range cur {
		if n == from {
			found = true
			next = append(next, to)
			continue
		}
		if n == to {
			continue
		}
		next = append(next, n)
	}
	if !found {
		return fmt.Errorf("%w: tab %s", ErrNotFound, from)
	}
	tools, err := List()
	if err != nil {
		return err
	}
	for _, t := range tools {
		if t.Group == from {
			t.Group = to
			if err := Save(t); err != nil {
				return err
			}
		}
	}
	return SaveTabs(next)
}

func RemoveTab(name, moveTo string) error {
	name = cleanTabName(name)
	if name == "" {
		return fmt.Errorf("%w: tab name required", ErrInvalid)
	}
	if name == DefaultOther {
		return fmt.Errorf("%w: cannot delete %s", ErrInvalid, DefaultOther)
	}
	moveTo = cleanTabName(moveTo)
	if moveTo == "" || moveTo == name {
		moveTo = DefaultOther
	}
	cur := LoadTabs()
	next := make([]string, 0, len(cur)+1)
	found := false
	for _, n := range cur {
		if n == name {
			found = true
			continue
		}
		next = append(next, n)
	}
	if !found {
		return fmt.Errorf("%w: tab %s", ErrNotFound, name)
	}
	next = withTab(next, moveTo)
	tools, err := List()
	if err != nil {
		return err
	}
	for _, t := range tools {
		if t.Group == name {
			t.Group = moveTo
			if err := Save(t); err != nil {
				return err
			}
		}
	}
	return SaveTabs(next)
}

func MoveTools(ids []string, group string) error {
	group = cleanTabName(group)
	if group == "" {
		group = DefaultOther
	}
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			cleaned = append(cleaned, id)
		}
	}
	if len(cleaned) == 0 {
		return fmt.Errorf("%w: ids required", ErrInvalid)
	}
	if err := EnsureTab(group); err != nil {
		return err
	}
	for _, id := range cleaned {
		t, err := Get(id)
		if err != nil {
			return err
		}
		t.Group = group
		if err := Save(t); err != nil {
			return err
		}
	}
	return nil
}

func ReplaceTabs(names []string) error {
	names = uniqueTabs(names)
	if len(names) == 0 {
		names = append([]string{}, DefaultTabs...)
	}
	allowed := map[string]struct{}{}
	for _, n := range names {
		allowed[n] = struct{}{}
	}
	if _, ok := allowed[DefaultOther]; !ok {
		names = append(names, DefaultOther)
		allowed[DefaultOther] = struct{}{}
	}
	tools, err := List()
	if err != nil {
		return err
	}
	for _, t := range tools {
		if _, ok := allowed[cleanTabName(t.Group)]; !ok {
			t.Group = DefaultOther
			if err := Save(t); err != nil {
				return err
			}
		}
	}
	return SaveTabs(names)
}
