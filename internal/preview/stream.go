package preview

import (
	"errors"
	"time"

	"github.com/markus/spidercam/internal/protocol"
)

const (
	DefaultWidth       = 1280
	DefaultHeight      = 720
	DefaultFPS         = 15
	DefaultBitrateKbps = 1200
)

type VideoFrame struct {
	RGBA   []byte
	Width  int
	Height int
}

type Config struct {
	Width        int
	Height       int
	FPS          int
	BitrateKbps  int
	Mock         bool
	MockKeyframe []byte
}

type subsample struct {
	n uint64
}

func (s *subsample) Tick() bool {
	s.n++
	return s.n%2 == 0
}

type Stream struct {
	cfg       Config
	enc       Encoder
	lastVideo string
	frameN    uint64
	sub       subsample
	out       chan []byte
	startTime time.Time
}

func New(cfg Config) (*Stream, error) {
	if cfg.Width <= 0 {
		cfg.Width = DefaultWidth
	}
	if cfg.Height <= 0 {
		cfg.Height = DefaultHeight
	}
	if cfg.FPS <= 0 {
		cfg.FPS = DefaultFPS
	}
	if cfg.BitrateKbps <= 0 {
		cfg.BitrateKbps = DefaultBitrateKbps
	}

	var enc Encoder
	var err error
	if cfg.Mock {
		if len(cfg.MockKeyframe) == 0 {
			return nil, errors.New("preview: mock mode requires MockKeyframe")
		}
		enc = newMockEncoder(cfg.MockKeyframe, cfg.FPS)
	} else {
		enc, err = openX264(cfg)
		if err != nil {
			return nil, err
		}
	}

	if enc != nil {
		enc.ForceKeyframe()
	}

	return &Stream{
		cfg:       cfg,
		enc:       enc,
		out:       make(chan []byte, 32),
		startTime: time.Now(),
	}, nil
}

func (s *Stream) ForceKeyframe() {
	if s.enc != nil {
		s.enc.ForceKeyframe()
	}
}

func (s *Stream) OnFrame(v VideoFrame, sel *protocol.SelectionState) bool {
	cut := false
	activeID := ""
	if sel != nil {
		activeID = sel.ActiveVideoID
	}
	if activeID != s.lastVideo {
		cut = true
		s.lastVideo = activeID
		s.enc.ForceKeyframe()
	}
	if !s.sub.Tick() {
		return cut
	}

	nextFrame := s.frameN + 1
	if nextFrame == 1 || nextFrame%uint64(s.cfg.FPS) == 0 {
		s.enc.ForceKeyframe()
	}

	ts := time.Now()
	result, err := s.enc.Encode(v.RGBA, v.Width, v.Height, ts)
	if err != nil || len(result.AVCC) == 0 {
		return cut
	}

	pts := uint64(time.Since(s.startTime).Microseconds())
	s.frameN++
	isKey := result.Keyframe || AvccIsKeyframe(result.AVCC)
	chunk := PackChunk(result.AVCC, pts, isKey)
	if isKey {
		s.out <- chunk
	} else {
		select {
		case s.out <- chunk:
		default:
		}
	}
	return cut
}

func (s *Stream) Chunks() <-chan []byte {
	return s.out
}

func (s *Stream) Seq() uint64 {
	return s.frameN
}

func (s *Stream) InitMessage() protocol.PreviewStreamInitMsg {
	return protocol.PreviewStreamInitMsg{
		Type:   "preview-stream-init",
		Codec:  DefaultCodec,
		Width:  s.cfg.Width,
		Height: s.cfg.Height,
		FPS:    s.cfg.FPS,
	}
}

func (s *Stream) Close() error {
	if s.enc == nil {
		return nil
	}
	return s.enc.Close()
}
