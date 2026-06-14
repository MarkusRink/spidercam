package scenario

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
)

const (
	tickInterval          = 20 * time.Millisecond
	loopDelayPublishEvery = 3 * time.Second
)

type StateListener func(state protocol.RoomState)
type ActiveVideoListener func(activeVideoID string, seq int)
type SelectionListener func()

type Engine struct {
	room              *room.Room
	tickCount         int
	previewSeq        int
	lastLoopDelayPub  time.Time
	lastActiveVideoID string
	audioDriven       bool

	mu                sync.Mutex
	stateListeners    []StateListener
	activeVideoListeners []ActiveVideoListener
	selectionListeners   []SelectionListener
}

func New(r *room.Room) *Engine {
	return &Engine{
		room:              r,
		lastActiveVideoID: r.ActiveVideoID(),
	}
}

func (e *Engine) Start(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.tick()
			}
		}
	}()
}

func (e *Engine) OnState(listener StateListener) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stateListeners = append(e.stateListeners, listener)
}

func (e *Engine) OnActiveVideoChange(listener ActiveVideoListener) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.activeVideoListeners = append(e.activeVideoListeners, listener)
}

func (e *Engine) OnSelectionChange(listener SelectionListener) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.selectionListeners = append(e.selectionListeners, listener)
}

func (e *Engine) RouteTo(id string) bool {
	var changed bool
	var activeID string
	ok := false
	e.room.UpdateState(func(s *protocol.RoomState) {
		if s.Selection == nil {
			return
		}
		valid := id == protocol.HostStreamID || participantExists(s.Participants, id)
		if !valid {
			return
		}
		ok = true
		prev := s.Selection.ActiveVideoID
		s.Selection.ActiveVideoID = id
		s.Selection.ActiveAudioID = id
		s.Selection.MainTalkerID = id
		s.Selection.Timestamp = time.Now().UnixMilli()
		activeID = id
		changed = prev != id
	})
	if !ok {
		return false
	}
	if changed {
		e.emitActiveVideoChange(activeID)
	}
	e.emitSelectionChange()
	e.emitState()
	return true
}

func (e *Engine) SetMixerState(mixerState protocol.MixerState) {
	e.room.UpdateState(func(s *protocol.RoomState) {
		if s.Selection == nil {
			return
		}
		s.Selection.MixerState = mixerState
		s.Selection.Timestamp = time.Now().UnixMilli()
	})
	e.emitSelectionChange()
	e.emitState()
}

func (e *Engine) SetAudioDriven(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.audioDriven = enabled
}

func (e *Engine) NotifyState() {
	e.emitState()
}

func (e *Engine) NotifySelectionChange() {
	e.emitSelectionChange()
}

func (e *Engine) NotifyActiveVideoIfChanged(activeVideoID string) {
	if activeVideoID != e.lastActiveVideoID {
		e.lastActiveVideoID = activeVideoID
		e.emitActiveVideoChange(activeVideoID)
	}
}

func (e *Engine) tick() {
	if e.audioDriven {
		e.tickAudioDriven()
		return
	}
	e.tickCount++
	now := time.Now()
	t := float64(e.tickCount) * 0.05

	e.room.UpdateState(func(s *protocol.RoomState) {
		for i := range s.Metrics {
			animateMetric(&s.Metrics[i], t)
			s.Metrics[i].LastUpdated = now.UnixMilli()
		}

		s.Reference.RmsDbfs = -42 + math.Sin(t*0.7)*4
		s.Reference.PeakDbfs = s.Reference.RmsDbfs + 5
		s.Reference.Vad = s.Reference.RmsDbfs > -40
		s.Reference.Active = s.Reference.Vad

		s.OutLevelDbfs = -24 + math.Sin(t*1.1)*6
		s.OutPeakDbfs = s.OutLevelDbfs + 4
		s.EnhancementBudgetPct = 3 + math.Sin(t*0.3)*1.5

		if s.Selection != nil {
			s.Selection.Timestamp = now.UnixMilli()
			if s.Selection.HoldRemainingMs > 0 {
				s.Selection.HoldRemainingMs -= int(tickInterval / time.Millisecond)
				if s.Selection.HoldRemainingMs < 0 {
					s.Selection.HoldRemainingMs = 0
				}
			}
		}

		if e.lastLoopDelayPub.IsZero() || now.Sub(e.lastLoopDelayPub) >= loopDelayPublishEvery {
			e.lastLoopDelayPub = now
			latency := 110 + int(math.Round(math.Sin(t)*8))
			s.GlobalLatencyMs = &latency
			for i := range s.Metrics {
				if s.Metrics[i].Role == protocol.StreamRoleParticipant && s.Metrics[i].LoopDelay.Known {
					ms := 95 + int(math.Round(math.Sin(t+1)*10))
					s.Metrics[i].LoopDelay = protocol.LoopDelayEstimate{
						Ms:            &ms,
						UncertaintyMs: 12,
						Known:         true,
					}
				}
			}
		}
	})

	activeVideoID := e.room.ActiveVideoID()
	if activeVideoID != e.lastActiveVideoID {
		e.lastActiveVideoID = activeVideoID
		e.emitActiveVideoChange(activeVideoID)
	}

	e.emitState()
}

func (e *Engine) tickAudioDriven() {
	e.room.UpdateState(func(s *protocol.RoomState) {
		if s.Selection != nil && s.Selection.HoldRemainingMs > 0 {
			s.Selection.HoldRemainingMs -= int(tickInterval / time.Millisecond)
			if s.Selection.HoldRemainingMs < 0 {
				s.Selection.HoldRemainingMs = 0
			}
		}
	})
	e.emitState()
}

func animateMetric(metric *protocol.StreamMetrics, t float64) {
	phase := float64(len(metric.ParticipantID)) + t
	base := -38.0
	amplitude := 2.0
	if metric.IsMainTalker {
		base = -22
		amplitude = 4
	}
	metric.RmsDbfs = base + math.Sin(phase*1.3)*amplitude
	metric.PeakDbfs = metric.RmsDbfs + 4 + math.Sin(phase*2.1)*1.5
	metric.SpeechLevelDbfs = metric.RmsDbfs - 2
	metric.SnrDb = 10 + math.Sin(phase*0.9)*6
	metric.Vad = metric.RmsDbfs > -30
	metric.Score = math.Max(0, math.Min(1, 0.3+math.Sin(phase)*0.2))
	metric.ScoreSmooth = metric.Score * 0.9
	if metric.Role == protocol.StreamRoleParticipant {
		rtt := 12 + math.Sin(phase)*3
		loss := math.Max(0, math.Sin(phase*0.5)*0.5)
		jitter := 8 + math.Sin(phase*1.7)*3
		bitrate := 300 + math.Sin(phase)*40
		fps := 29 + math.Sin(phase*0.4)
		metric.RttMs = &rtt
		metric.PacketLoss = &loss
		metric.JitterMs = &jitter
		metric.BitrateKbps = &bitrate
		metric.FramesPerSecond = &fps
	}
}

func (e *Engine) emitState() {
	state := e.room.FullState(nil)
	e.mu.Lock()
	listeners := append([]StateListener(nil), e.stateListeners...)
	e.mu.Unlock()
	for _, listener := range listeners {
		listener(state)
	}
}

func (e *Engine) emitActiveVideoChange(activeVideoID string) {
	e.mu.Lock()
	e.previewSeq++
	seq := e.previewSeq
	listeners := append([]ActiveVideoListener(nil), e.activeVideoListeners...)
	e.mu.Unlock()
	for _, listener := range listeners {
		listener(activeVideoID, seq)
	}
}

func (e *Engine) emitSelectionChange() {
	e.mu.Lock()
	listeners := append([]SelectionListener(nil), e.selectionListeners...)
	e.mu.Unlock()
	for _, listener := range listeners {
		listener()
	}
}

func participantExists(participants []protocol.ParticipantInfo, id string) bool {
	for _, p := range participants {
		if p.ID == id {
			return true
		}
	}
	return false
}

func ParseMixerState(value string) (protocol.MixerState, bool) {
	switch strings.ToUpper(value) {
	case string(protocol.MixerLocked):
		return protocol.MixerLocked, true
	case string(protocol.MixerHold):
		return protocol.MixerHold, true
	case string(protocol.MixerSwitch):
		return protocol.MixerSwitch, true
	case string(protocol.MixerSilence):
		return protocol.MixerSilence, true
	default:
		return "", false
	}
}
