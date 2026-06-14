package audio

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/markus/spidercam/internal/audio/mixer"
	"github.com/markus/spidercam/internal/audio/processor"
	"github.com/markus/spidercam/internal/audio/reference"
	"github.com/markus/spidercam/internal/output"
	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
	"github.com/markus/spidercam/internal/scenario"
)

type MockCapture struct {
	engine    *Engine
	ref       *reference.Processor
	phase     float64
	mu        sync.Mutex
	streamIDs []string
}

func NewMockCapture(engine *Engine, ref *reference.Processor, streamIDs []string) *MockCapture {
	return &MockCapture{
		engine:    engine,
		ref:       ref,
		streamIDs: append([]string(nil), streamIDs...),
	}
}

func (m *MockCapture) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.pushFrame()
			}
		}
	}()
}

func (m *MockCapture) pushFrame() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.phase += 0.01

	refAmp := 0.05 + 0.04*math.Sin(m.phase*0.7)
	m.ref.PushFrame(SineFrame(refAmp, 220, m.phase))

	for i, id := range m.streamIDs {
		p := m.engine.Stream(id)
		if p == nil {
			continue
		}
		freq := 300.0 + float64(i)*110
		amp := 0.02 + 0.15*math.Max(0, math.Sin(m.phase*1.3+float64(i)*2.1))
		if id == protocol.HostStreamID {
			amp *= 0.6
		}
		p.Jitter().Push(SineFrame(amp, freq, m.phase+float64(i)))
	}
}

type Bridge struct {
	engine   *Engine
	room     *room.Room
	scenario *scenario.Engine
	output   output.Writer
}

func NewBridge(engine *Engine, rm *room.Room, sc *scenario.Engine, w output.Writer) *Bridge {
	return &Bridge{
		engine:   engine,
		room:     rm,
		scenario: sc,
		output:   w,
	}
}

func (b *Bridge) Start(ctx context.Context) {
	go b.engine.Run(ctx, func(frame mixer.Frame) {
		if b.output != nil {
			_ = b.output.WritePCM(frame.PCM)
		}
		if b.engine.TickCount()%2 != 0 {
			return
		}
		b.publishRoom(frame)
	})
}

func (b *Bridge) publishRoom(frame mixer.Frame) {
	metrics := b.engine.Metrics()
	sel := b.engine.Selection()
	ref := b.engine.ReferenceMetrics()
	delays := b.engine.DelayEstimates()
	global := reference.GlobalLatency(delays)

	for i := range metrics {
		if est, ok := delays[metrics[i].ParticipantID]; ok {
			metrics[i].LoopDelay = est
		}
	}

	b.room.UpdateState(func(s *protocol.RoomState) {
		s.Metrics = metrics
		s.Reference = ref
		selCopy := sel
		s.Selection = &selCopy
		s.OutLevelDbfs = frame.OutDbfs
		s.OutPeakDbfs = frame.OutPeakDbfs
		s.EnhancementBudgetPct = b.engine.EnhancementBudgetPct()
		s.GlobalLatencyMs = global
		if b.output != nil {
			s.OutputHealthy = b.output.Healthy()
		}
	})

	if b.scenario != nil {
		activeVideo := sel.ActiveVideoID
		b.scenario.NotifyActiveVideoIfChanged(activeVideo)
		b.scenario.NotifySelectionChange()
		b.scenario.NotifyState()
	}
}

func SetupMockAudio(ctx context.Context, rm *room.Room, sc *scenario.Engine) (*Engine, *MockCapture, output.Writer) {
	cfg := rm.Config()
	engine := NewEngine(cfg)
	ref := engine.reference

	host := processor.NewPipeline(protocol.HostStreamID, "Host", protocol.StreamRoleHost)
	engine.AttachStream(host)

	state := rm.State()
	for _, p := range state.Participants {
		role := protocol.StreamRoleParticipant
		pipe := processor.NewPipeline(p.ID, p.Name, role)
		engine.AttachStream(pipe)
	}

	for _, m := range state.Metrics {
		if p := engine.Stream(m.ParticipantID); p != nil {
			flags := protocol.StreamProcessingFlags{
				AecEnabled:     m.AecEnabled,
				DenoiseEnabled: m.DenoiseEnabled,
			}
			_ = p.SetProcessing(flags)
		}
	}

	ids := []string{protocol.HostStreamID}
	for _, p := range state.Participants {
		ids = append(ids, p.ID)
	}

	capture := NewMockCapture(engine, ref, ids)
	capture.Start(ctx)

	writer := output.NewMockWriter()
	bridge := NewBridge(engine, rm, sc, writer)
	bridge.Start(ctx)

	return engine, capture, writer
}
