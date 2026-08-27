package registry

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// GitWebURL turns a git remote into an http(s) page URL. Rejects javascript/data.
func GitWebURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "file:") {
		return ""
	}
	raw = strings.TrimSuffix(raw, ".git")
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return ""
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return ""
		}
		u.User = nil
		return u.String()
	}
	if strings.HasPrefix(lower, "ssh://") {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return ""
		}
		p := strings.TrimSuffix(u.Path, ".git")
		return "https://" + u.Host + p
	}
	if len(raw) >= 4 && strings.EqualFold(raw[:4], "git@") {
		rest := raw[4:]
		rest = strings.TrimSuffix(rest, ".git")
		host, repo, ok := strings.Cut(rest, ":")
		if !ok || host == "" || strings.TrimSpace(repo) == "" {
			return ""
		}
		return "https://" + host + "/" + strings.TrimPrefix(repo, "/")
	}
	return ""
}

// GitLabel is a short owner/repo for the UI.
func GitLabel(raw string) string {
	web := GitWebURL(raw)
	if web == "" {
		return strings.TrimSpace(raw)
	}
	u, err := url.Parse(web)
	if err != nil {
		return web
	}
	p := strings.Trim(u.Path, "/")
	p = strings.TrimSuffix(p, ".git")
	base := path.Base(p)
	dir := path.Base(path.Dir(p))
	if dir != "" && dir != "." && dir != "/" && base != "" && base != "." {
		return dir + "/" + base
	}
	if base != "" && base != "." {
		return base
	}
	return u.Host
}

// ScrubGit removes userinfo from http(s) remotes so credentials never persist.
func ScrubGit(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	switch u.Scheme {
	case "http", "https", "ssh":
		if u.User == nil {
			return raw
		}
		u.User = nil
		return u.String()
	default:
		return raw
	}
}

func LoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// CheckHTTPURL rejects javascript:/file:/data: and anything that is not http(s).
func CheckHTTPURL(raw, field string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%w: %s is not a valid http(s) URL", ErrInvalid, field)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("%w: %s scheme %q not allowed", ErrInvalid, field, u.Scheme)
	}
}

// CheckLocalHTTPURL is for health_url: http(s) on loopback only (no SSRF).
func CheckLocalHTTPURL(raw, field string) error {
	if err := CheckHTTPURL(raw, field); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || !LoopbackHost(u.Hostname()) {
		return fmt.Errorf("%w: %s must be localhost / 127.0.0.1", ErrInvalid, field)
	}
	return nil
}
