package capture_test

import (
	"context"
	"testing"

	"github.com/markus/spidercam/internal/capture"
)

func TestListDevicesStub(t *testing.T) {
	devs, err := capture.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devs.Mics) == 0 {
		t.Fatal("expected mock mics")
	}
	if len(devs.Sinks) == 0 {
		t.Fatal("expected mock sinks")
	}
	if len(devs.Cameras) == 0 {
		t.Fatal("expected mock cameras")
	}
}

func TestOpenReadCloseStub(t *testing.T) {
	ctx := context.Background()
	b, err := capture.Open(ctx, capture.Selection{}, capture.DefaultSampleRate)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	mic := make([]float32, capture.FrameSamples)
	n := b.ReadMic(mic)
	if n != capture.FrameSamples {
		t.Fatalf("ReadMic samples = %d, want %d", n, capture.FrameSamples)
	}

	mon := make([]float32, capture.FrameSamples)
	n = b.ReadMonitor(mon)
	if n != capture.FrameSamples {
		t.Fatalf("ReadMonitor samples = %d, want %d", n, capture.FrameSamples)
	}

	if _, _, _, ok := b.ReadCamera(); ok {
		t.Fatal("stub camera should not produce frames")
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if b.ReadMic(mic) != 0 {
		t.Fatal("ReadMic after Close should return 0")
	}
}

func TestReopenStub(t *testing.T) {
	ctx := context.Background()
	b, err := capture.Open(ctx, capture.Selection{}, capture.DefaultSampleRate)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer b.Close()

	sel := capture.Selection{
		MicID:    "mock-mic",
		CameraID: "mock-cam",
		SinkID:   "mock-sink",
	}
	if err := b.Reopen(ctx, sel); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	state := b.ActiveState()
	if state.MicID != sel.MicID {
		t.Fatalf("MicID = %q, want %q", state.MicID, sel.MicID)
	}
	if state.MicLabel == "" {
		t.Fatal("expected mic label after Reopen")
	}
}

func TestSelectionFromEnv(t *testing.T) {
	t.Setenv("SPIDERCAM_MIC", "mic-1")
	t.Setenv("SPIDERCAM_CAMERA", "/dev/video0")
	t.Setenv("SPIDERCAM_PLAYBACK_SINK", "sink-1")

	sel := capture.SelectionFromEnv()
	if sel.MicID != "mic-1" || sel.CameraID != "/dev/video0" || sel.SinkID != "sink-1" {
		t.Fatalf("unexpected selection: %+v", sel)
	}
}
