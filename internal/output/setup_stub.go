//go:build !linux

package output

import "fmt"

func SetupLoopback() (string, error) {
	return "", fmt.Errorf("virtual camera setup requires linux")
}
