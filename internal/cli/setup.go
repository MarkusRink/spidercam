package cli

import (
	"flag"
	"log"

	"github.com/markus/spidercam/internal/output"
)

func runSetup(args []string) int {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	_ = fs.Parse(args)

	path, err := output.SetupLoopback()
	if err != nil {
		log.Print(err)
		return 1
	}
	log.Printf("virtual camera ready: %s", path)
	log.Print("virtual microphone is created automatically when spidercamd starts")
	return 0
}
