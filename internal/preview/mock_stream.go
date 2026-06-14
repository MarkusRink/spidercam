package preview

import (
	"context"
	"sync"
	"time"

	"github.com/markus/spidercam/internal/protocol"
)

const (
	MockCodec  = DefaultCodec
	MockWidth  = 1280
	MockHeight = 720
	MockFPS    = 15
)

type ChunkListener func(chunk []byte)
type CutListener func(activeVideoID string, seq int)

type MockStream struct {
	avcc []byte

	mu             sync.Mutex
	chunkListeners map[int]ChunkListener
	cutListeners   map[int]CutListener
	nextListenerID int
	frameIndex     int
	startTime      time.Time
	forceKeyframe  bool
}

func NewMockStream(annexB []byte) *MockStream {
	return &MockStream{
		avcc:           AnnexBToAVCC(annexB),
		chunkListeners: make(map[int]ChunkListener),
		cutListeners:   make(map[int]CutListener),
		forceKeyframe:  true,
	}
}

func (s *MockStream) Start(ctx context.Context) {
	s.mu.Lock()
	s.startTime = time.Now()
	s.mu.Unlock()

	interval := time.Second / MockFPS
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.emitFrame()
			}
		}
	}()
}

func (s *MockStream) OnChunk(listener ChunkListener) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextListenerID
	s.nextListenerID++
	s.chunkListeners[id] = listener
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.chunkListeners, id)
	}
}

func (s *MockStream) OnCut(listener CutListener) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextListenerID
	s.nextListenerID++
	s.cutListeners[id] = listener
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.cutListeners, id)
	}
}

func (s *MockStream) NotifyCut(activeVideoID string, seq int) {
	s.mu.Lock()
	s.forceKeyframe = true
	listeners := make([]CutListener, 0, len(s.cutListeners))
	for _, l := range s.cutListeners {
		listeners = append(listeners, l)
	}
	s.mu.Unlock()
	for _, listener := range listeners {
		listener(activeVideoID, seq)
	}
}

func (s *MockStream) InitMessage() protocol.PreviewStreamInitMsg {
	return protocol.PreviewStreamInitMsg{
		Type:   "preview-stream-init",
		Codec:  MockCodec,
		Width:  MockWidth,
		Height: MockHeight,
		FPS:    MockFPS,
	}
}

func (s *MockStream) emitFrame() {
	s.mu.Lock()
	ptsUs := uint64(time.Since(s.startTime).Microseconds()) + uint64(s.frameIndex)*(1_000_000/MockFPS)
	keyframe := s.forceKeyframe || s.frameIndex%MockFPS == 0
	s.forceKeyframe = false
	s.frameIndex++
	avcc := s.avcc
	listeners := make([]ChunkListener, 0, len(s.chunkListeners))
	for _, l := range s.chunkListeners {
		listeners = append(listeners, l)
	}
	s.mu.Unlock()

	chunk := PackChunk(avcc, ptsUs, keyframe)
	for _, listener := range listeners {
		listener(chunk)
	}
}
