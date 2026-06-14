//go:build linux

package output

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

func FindLoopbackDevice() (string, error) {
	entries, err := os.ReadDir("/sys/class/video4linux")
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		namePath := "/sys/class/video4linux/" + e.Name() + "/name"
		data, err := os.ReadFile(namePath)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(data))
		if strings.Contains(strings.ToLower(name), "loopback") {
			return "/dev/" + e.Name(), nil
		}
	}
	return "", errors.New("no v4l2loopback device found")
}

func V4L2LoopbackLoaded() bool {
	f, err := os.Open("/proc/modules")
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "v4l2loopback ") {
			return true
		}
	}
	return false
}

func deviceCardName(path string) string {
	base := strings.TrimPrefix(path, "/dev/")
	data, err := os.ReadFile("/sys/class/video4linux/" + base + "/name")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
