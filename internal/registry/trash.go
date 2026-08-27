package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const TrashTab = "垃圾桶"

var ErrBlocked = errors.New("tool blocked")

// BlockedEntry is a removed tool that must not be registered again
// until Restore or Unblock.
type BlockedEntry struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Workdir  string    `json:"workdir,omitempty"`
	Git      string    `json:"git,omitempty"`
	URL      string    `json:"url,omitempty"`
	BuriedAt time.Time `json:"buried_at"`
	Tool     Tool      `json:"tool"`
}

func trashDir() (string, error) {
	root, err := RootDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "trash")
	return dir, os.MkdirAll(dir, 0o755)
}

func trashPath(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("%w: id %q", ErrInvalid, id)
	}
	dir, err := trashDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, id+".json")
	rel, err := filepath.Rel(dir, p)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: id %q", ErrInvalid, id)
	}
	return p, nil
}

func normWorkdir(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return strings.TrimRight(filepath.Clean(s), `/\`)
}

func workdirEq(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return false
}

func gitKey(raw string) string {
	return strings.ToLower(strings.TrimSpace(GitWebURL(raw)))
}

func urlKey(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	host := strings.ToLower(u.Host)
	path := strings.TrimRight(u.Path, "/")
	return strings.ToLower(u.Scheme) + "://" + host + path
}

func idEq(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return false
}

func CheckBlocked(t Tool) error {
	entries, err := ListBlocked()
	if err != nil {
		return err
	}
	nid := strings.TrimSpace(t.ID)
	nw := normWorkdir(t.Workdir)
	ng := gitKey(t.Git)
	nu := urlKey(t.URL)
	for _, e := range entries {
		if idEq(e.ID, nid) {
			return fmt.Errorf("%w: 「%s」在垃圾桶里，不会再注册", ErrBlocked, displayName(e))
		}
		if nw != "" && workdirEq(nw, e.Workdir) {
			return fmt.Errorf("%w: 这个目录已屏蔽（曾移除 %s）", ErrBlocked, displayName(e))
		}
		if ng != "" && e.Git != "" && ng == e.Git {
			return fmt.Errorf("%w: 这个仓库已屏蔽（曾移除 %s）", ErrBlocked, displayName(e))
		}
		if nu != "" && e.URL != "" && nu == e.URL {
			return fmt.Errorf("%w: 这个网址已屏蔽（曾移除 %s）", ErrBlocked, displayName(e))
		}
	}
	return nil
}

func displayName(e BlockedEntry) string {
	if strings.TrimSpace(e.Name) != "" {
		return e.Name
	}
	return e.ID
}

func Bury(t Tool) error {
	t = Normalize(t)
	if !idPattern.MatchString(t.ID) {
		return fmt.Errorf("%w: id %q", ErrInvalid, t.ID)
	}
	entry := BlockedEntry{
		ID:       t.ID,
		Name:     t.Name,
		Workdir:  normWorkdir(t.Workdir),
		Git:      gitKey(t.Git),
		URL:      urlKey(t.URL),
		BuriedAt: time.Now().UTC(),
		Tool:     t,
	}
	p, err := trashPath(t.ID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}

func ListBlocked() ([]BlockedEntry, error) {
	dir, err := trashDir()
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]BlockedEntry, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var b BlockedEntry
		if json.Unmarshal(data, &b) != nil || b.ID == "" {
			continue
		}
		b.Workdir = normWorkdir(b.Workdir)
		if b.Git != "" {
			b.Git = gitKey(b.Git)
		}
		if b.URL != "" {
			b.URL = urlKey(b.URL)
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func Restore(id string) error {
	entry, err := readTrash(id)
	if err != nil {
		return err
	}
	if _, err := Get(id); err == nil {
		return fmt.Errorf("%w: %s 已在工具箱", ErrInvalid, id)
	}
	if err := dropTrash(id); err != nil {
		return err
	}
	if err := saveTool(entry.Tool, false); err != nil {
		_ = Bury(entry.Tool)
		return err
	}
	return nil
}

func Unblock(id string) error {
	return dropTrash(id)
}

func readTrash(id string) (BlockedEntry, error) {
	p, err := trashPath(id)
	if err != nil {
		return BlockedEntry{}, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return BlockedEntry{}, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return BlockedEntry{}, err
	}
	var b BlockedEntry
	if err := json.Unmarshal(data, &b); err != nil {
		return BlockedEntry{}, err
	}
	if b.Tool.ID == "" {
		b.Tool.ID = b.ID
	}
	return b, nil
}

func dropTrash(id string) error {
	p, err := trashPath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return err
	}
	return nil
}
