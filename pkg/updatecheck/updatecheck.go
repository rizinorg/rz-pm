// Package updatecheck provides helpers for periodic update
// checks using a timestamp file.
package updatecheck

import (
	"os"
	"strings"
	"time"
)

// Checker manages update check timestamps in a file.
type Checker struct {
	Path     string
	Interval time.Duration
}

// ShouldCheck returns true if enough time has passed since the last check,
// or if no timestamp exists or cannot be parsed.
func (c *Checker) ShouldCheck() (bool, error) {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		// If file doesn't exist, check for update
		return true, nil
	}
	lastCheck, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		// If parse fails, check for update
		return true, nil
	}
	return time.Since(lastCheck) > c.Interval, nil
}

// UpdateTimestamp writes the current time to the file, creating parent directories if needed.
func (c *Checker) UpdateTimestamp() error {
	dir := getDir(c.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	return os.WriteFile(c.Path, []byte(now), 0o644)
}

// getDir returns the directory part of the path.
func getDir(path string) string {
	lastSep := strings.LastIndexAny(path, "/\\")
	if lastSep == -1 {
		return "."
	}
	return path[:lastSep]
}
