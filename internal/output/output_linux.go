//go:build linux

package output

import (
	"context"
	"fmt"
)

type platformWriter struct {
	video *v4l2Writer
	audio *pulseSink
}

func (w *platformWriter) WritePCM(samples []float32) error {
	return w.audio.WritePCM(samples)
}

func (w *platformWriter) WriteVideo(rgba []byte, width, height int) error {
	return w.video.WriteVideo(rgba, width, height)
}

func (w *platformWriter) Healthy() bool {
	return w.video.Healthy() && w.audio.Healthy()
}

func (w *platformWriter) Close() error {
	var errs []error
	if w.video != nil {
		errs = append(errs, w.video.Close())
	}
	if w.audio != nil {
		errs = append(errs, w.audio.Close())
	}
	return joinErrors(errs)
}

func openPlatform(ctx context.Context, cfg Config) (Writer, error) {
	_ = ctx

	vidPath := cfg.VideoDevice
	if vidPath == "" {
		var err error
		vidPath, err = FindLoopbackDevice()
		if err != nil {
			return nil, missingVideoDeviceErr("no v4l2loopback device", err)
		}
	}

	video, err := openV4L2Output(vidPath, cfg.Width, cfg.Height)
	if err != nil {
		return nil, missingVideoDeviceErr(fmt.Sprintf("video %s", vidPath), err)
	}

	audio, err := openPulseSink(cfg.AudioSink)
	if err != nil {
		_ = video.Close()
		return nil, fmt.Errorf("audio output: %w", err)
	}

	return &platformWriter{video: video, audio: audio}, nil
}

func joinErrors(errs []error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
