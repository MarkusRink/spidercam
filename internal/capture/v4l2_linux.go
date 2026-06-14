//go:build linux

package capture

import (
	"os"
	"path/filepath"
	"strings"
)

func readCardName(devPath string) string {
	base := filepath.Base(devPath)
	data, err := os.ReadFile(filepath.Join("/sys/class/video4linux", base, "name"))
	if err != nil {
		return "(unknown)"
	}
	return strings.TrimSpace(string(data))
}

func filepathGlob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
