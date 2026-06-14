package output

import "testing"

func TestMockWriterWritePCM(t *testing.T) {
	w := NewMockWriter()
	frame := make([]float32, FrameSamples)
	frame[0] = 0.5
	if err := w.WritePCM(frame); err != nil {
		t.Fatal(err)
	}
	if !w.Healthy() {
		t.Fatal("mock writer should be healthy")
	}
	if w.FramesWritten() != 1 {
		t.Fatalf("frames = %d", w.FramesWritten())
	}
	last := w.LastFrame()
	if len(last) != FrameSamples || last[0] != 0.5 {
		t.Fatalf("last frame mismatch")
	}
}
