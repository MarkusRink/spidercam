package output

import (
	"context"
	"fmt"
)

func Open(ctx context.Context, cfg Config) (Writer, error) {
	if cfg.Mock {
		return NewMockWriter(), nil
	}
	if cfg.Width <= 0 {
		cfg.Width = DefaultWidth
	}
	if cfg.Height <= 0 {
		cfg.Height = DefaultHeight
	}
	if cfg.AudioSink == "" {
		cfg.AudioSink = DefaultAudioSink
	}
	return openPlatform(ctx, cfg)
}

func missingVideoDeviceErr(pathHint string, err error) error {
	return fmt.Errorf(
		"%s: %w\n\nVirtual camera setup (once per machine):\n  %s\n  %s",
		pathHint,
		err,
		LoopbackSetup,
		SetupSubcommand,
	)
}
