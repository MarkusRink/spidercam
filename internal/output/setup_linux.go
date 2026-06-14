//go:build linux

package output

import (
	"fmt"
	"os/exec"
)

func SetupLoopback() (string, error) {
	if V4L2LoopbackLoaded() {
		if path, err := FindLoopbackDevice(); err == nil {
			return path, nil
		}
	}

	if err := loadLoopbackModule(); err != nil {
		return "", fmt.Errorf("load v4l2loopback: %w\n\nTry:\n  %s\n  %s", err, LoopbackSetup, SetupSubcommand)
	}

	path, err := FindLoopbackDevice()
	if err != nil {
		return "", fmt.Errorf("v4l2loopback loaded but no loopback device found: %w", err)
	}
	return path, nil
}

func loadLoopbackModule() error {
	args := append([]string{LoopbackModprobeModule}, LoopbackModprobeOptions...)
	if err := exec.Command("modprobe", args...).Run(); err == nil {
		return nil
	}
	return exec.Command("sudo", append([]string{"modprobe"}, args...)...).Run()
}
