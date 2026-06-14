package output

import (
	"sync"
)

type MockWriter struct {
	mu      sync.Mutex
	healthy bool
	frames  int
	last    []float32
}

func NewMockWriter() *MockWriter {
	return &MockWriter{healthy: true}
}

func (w *MockWriter) WritePCM(samples []float32) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.frames++
	w.last = append(w.last[:0], samples...)
	return nil
}

func (w *MockWriter) WriteVideo([]byte, int, int) error {
	return nil
}

func (w *MockWriter) Healthy() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.healthy
}

func (w *MockWriter) SetHealthy(ok bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.healthy = ok
}

func (w *MockWriter) FramesWritten() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.frames
}

func (w *MockWriter) LastFrame() []float32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]float32, len(w.last))
	copy(out, w.last)
	return out
}

func (w *MockWriter) Close() error {
	return nil
}
