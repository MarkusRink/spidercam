package output

import (
	"context"
	"testing"
)

func TestOpenMock(t *testing.T) {
	w, err := Open(context.Background(), Config{Mock: true})
	if err != nil {
		t.Fatal(err)
	}
	if !w.Healthy() {
		t.Fatal("mock writer should be healthy")
	}
	if err := w.WritePCM(make([]float32, FrameSamples)); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteVideo(nil, 1280, 720); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.AudioSink != DefaultAudioSink {
		t.Fatalf("audio sink = %q", cfg.AudioSink)
	}
	if cfg.Width != DefaultWidth || cfg.Height != DefaultHeight {
		t.Fatalf("dimensions = %dx%d", cfg.Width, cfg.Height)
	}
}
