//go:build linux

package daemon

import (
	"log"
	"os/exec"
)

func openBrowser(url string) {
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		log.Printf("open browser: %v", err)
		return
	}
	log.Printf("opened host UI in browser (disable with --no-open-browser)")
}
