package main

import (
	"os"

	"github.com/markus/spidercam/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
