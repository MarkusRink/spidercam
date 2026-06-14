//go:build !linux

package output

import (
	"context"
	"fmt"
)

func openPlatform(ctx context.Context, cfg Config) (Writer, error) {
	_ = ctx
	_ = cfg
	return nil, fmt.Errorf("virtual output requires linux\n\nVirtual camera setup:\n  %s\n  %s", LoopbackSetup, SetupSubcommand)
}
