package docs

import (
	"os"
	"path/filepath"

	"github.com/ts721521/DevToolbox/internal/registry"
)

func Publish(files map[string][]byte) error {
	root, err := registry.RootDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
