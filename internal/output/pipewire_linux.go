//go:build linux

package output

import (
	"os/exec"
)

func refreshCameraSourceSync() error {
	return exec.Command("systemctl", "--user", "restart", "wireplumber").Run()
}

func primeV4L2Output(video *v4l2Writer, width, height int) error {
	rgba := make([]byte, width*height*4)
	for i := 0; i < len(rgba)/4; i++ {
		rgba[i*4+0] = 255
		rgba[i*4+3] = 255
	}
	for i := 0; i < 3; i++ {
		if err := video.WriteVideo(rgba, width, height); err != nil {
			return err
		}
	}
	return nil
}
