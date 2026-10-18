package contentmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteFile writes content.md into an issue directory.
func WriteFile(dir, markdown string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("empty issue directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, ContentFileName)
	return os.WriteFile(path, []byte(markdown), 0o644)
}
