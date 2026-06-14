//go:build cgo && linux && spidercam_native_capture

package capture_test

import (
	"context"
	"testing"
	"time"

	"github.com/markus/spidercam/internal/capture"
)

func TestReopenUnder500ms(t *testing.T) {
	devs, err := capture.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devs.Sinks) == 0 {
		t.Skip("no pipewire sinks")
	}

	sinkA := devs.Sinks[0].ID
	sinkB := sinkA
	if len(devs.Sinks) >= 2 {
		sinkB = devs.Sinks[1].ID
	}

	ctx := context.Background()
	b, err := capture.Open(ctx, capture.Selection{SinkID: sinkA}, capture.DefaultSampleRate)
	if err != nil {
		t.Fatalf("Open sink A: %v", err)
	}

	mon := make([]float32, capture.FrameSamples)
	for i := 0; i < 5; i++ {
		_ = b.ReadMonitor(mon)
		time.Sleep(10 * time.Millisecond)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	start := time.Now()
	b, err = capture.Open(ctx, capture.Selection{SinkID: sinkB}, capture.DefaultSampleRate)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Reopen sink B: %v", err)
	}
	defer b.Close()

	if elapsed >= 500*time.Millisecond {
		t.Fatalf("reopen took %v, want < 500ms", elapsed)
	}

	for i := 0; i < 5; i++ {
		_ = b.ReadMonitor(mon)
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBundleReopenSinkSwitch(t *testing.T) {
	devs, err := capture.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devs.Sinks) == 0 {
		t.Skip("no pipewire sinks")
	}

	sinkA := devs.Sinks[0].ID
	sinkB := sinkA
	if len(devs.Sinks) >= 2 {
		sinkB = devs.Sinks[1].ID
	}

	ctx := context.Background()
	b, err := capture.Open(ctx, capture.Selection{SinkID: sinkA}, capture.DefaultSampleRate)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	start := time.Now()
	if err := b.Reopen(ctx, capture.Selection{SinkID: sinkB}); err != nil {
		b.Close()
		t.Fatalf("Reopen: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed >= 500*time.Millisecond {
		b.Close()
		t.Fatalf("Reopen took %v, want < 500ms", elapsed)
	}

	mon := make([]float32, capture.FrameSamples)
	for i := 0; i < 5; i++ {
		_ = b.ReadMonitor(mon)
		time.Sleep(10 * time.Millisecond)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
