package config

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
)

//go:embed example.yaml
var starter []byte

// Init writes the starter config (Alice, Bob, Carol, and the built-in
// provider profiles) to path. An existing file is left alone unless force
// is set. An empty path means DefaultPath in the current directory.
func Init(path string, force bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultPath
	}
	if !force {
		_, err := os.Stat(path)
		switch {
		case err == nil:
			return fmt.Errorf("%s already exists (pass --force to replace it)", path)
		case !os.IsNotExist(err):
			return err
		}
	}
	if err := os.WriteFile(path, starter, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
