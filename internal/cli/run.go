package cli

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/markus/spidercam/internal/daemon"
)

func Run(args []string) int {
	if len(args) > 0 && args[0] == "setup" {
		return runSetup(args[1:])
	}

	fs := flag.NewFlagSet("spidercamd", flag.ExitOnError)
	noOpen := fs.Bool("no-open-browser", false, "do not open host UI in browser")
	hostAddr := fs.String("host-addr", "127.0.0.1:1235", "host UI bind address")
	participantAddr := fs.String("participant-addr", "0.0.0.0:1234", "participant UI bind address")
	mock := fs.Bool("mock", false, "mock capture and output (dev/CI)")
	_ = fs.Parse(args)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := daemon.LoadConfig(*hostAddr, *participantAddr, *mock, !*noOpen)
	if err := daemon.Run(ctx, cfg); err != nil {
		log.Print(err)
		return 1
	}
	return 0
}
