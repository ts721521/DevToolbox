package applog

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/ts721521/DevToolbox/internal/registry"
)

const (
	fileName = "tooldock.log"
	maxBytes = 2 << 20 // 2MB
)

var logger *slog.Logger

func Dir() (string, error) {
	root, err := registry.RootDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "logs")
	return dir, os.MkdirAll(dir, 0o755)
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func Init() error {
	p, err := Path()
	if err != nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
		slog.SetDefault(logger)
		return err
	}
	f, err := newRotating(p, maxBytes)
	if err != nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
		slog.SetDefault(logger)
		return err
	}
	w := io.MultiWriter(os.Stderr, f)
	logger = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	return nil
}

func Info(msg string, args ...any) {
	ensure().Info(msg, args...)
}

func Warn(msg string, args ...any) {
	ensure().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	ensure().Error(msg, args...)
}

func ensure() *slog.Logger {
	if logger == nil {
		_ = Init()
	}
	if logger == nil {
		return slog.Default()
	}
	return logger
}

func Tail(n int) (string, error) {
	if n <= 0 {
		n = 200
	}
	if n > 2000 {
		n = 2000
	}
	p, err := Path()
	if err != nil {
		return "", err
	}
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
		if len(lines) > n {
			lines = lines[len(lines)-n:]
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += "\n"
		}
		out += line
	}
	if out != "" {
		out += "\n"
	}
	return out, nil
}

type rotating struct {
	mu   sync.Mutex
	path string
	max  int64
	f    *os.File
}

func newRotating(path string, max int64) (*rotating, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &rotating{path: path, max: max, f: f}, nil
}

func (r *rotating) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f != nil {
		if st, err := r.f.Stat(); err == nil && st.Size() >= r.max {
			_ = r.f.Close()
			bak := r.path + ".1"
			_ = os.Remove(bak)
			_ = os.Rename(r.path, bak)
			nf, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
			if err != nil {
				r.f = nil
				return 0, err
			}
			r.f = nf
		}
	}
	if r.f == nil {
		return 0, fmt.Errorf("log file closed")
	}
	return r.f.Write(p)
}
