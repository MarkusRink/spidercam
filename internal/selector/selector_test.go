package selector

import (
	"testing"
	"time"

	"github.com/markus/spidercam/internal/protocol"
)

func defaultCfg() protocol.HostConfig {
	return protocol.DefaultHostConfig
}

func metric(id string, scoreSmooth float64, vad bool) protocol.StreamMetrics {
	return protocol.StreamMetrics{
		ParticipantID: id,
		Name:          id,
		Role:          protocol.StreamRoleParticipant,
		ScoreSmooth:   scoreSmooth,
		Vad:           vad,
		AudioActive:   true,
		VideoActive:   true,
	}
}

func hostMetric(scoreSmooth float64, vad bool) protocol.StreamMetrics {
	m := metric(protocol.HostStreamID, scoreSmooth, vad)
	m.Role = protocol.StreamRoleHost
	m.Name = "Host"
	return m
}

func TestSilenceWhenBelowThreshold(t *testing.T) {
	cfg := defaultCfg()
	state := NewState(cfg)
	streams := []protocol.StreamMetrics{
		metric("alice", 0.05, false),
		metric("bob", 0.08, false),
	}
	sel := Select(state, cfg, streams, time.Now())
	if sel.MixerState != protocol.MixerSilence {
		t.Fatalf("mixer state = %v, want SILENCE", sel.MixerState)
	}
}

func TestRefExcluded(t *testing.T) {
	cfg := defaultCfg()
	state := NewState(cfg)
	ref := metric(protocol.PlaybackRefStreamID, 0.99, true)
	ref.Role = protocol.StreamRoleReference
	streams := []protocol.StreamMetrics{ref, metric("alice", 0.2, true)}
	sel := Select(state, cfg, streams, time.Now())
	if sel.MainTalkerID == protocol.PlaybackRefStreamID {
		t.Fatal("playback-ref must not be selected")
	}
}

func TestMarginAndHold(t *testing.T) {
	cfg := defaultCfg()
	cfg.SwitchMargin = 0.1
	cfg.AudioHoldMs = 400
	cfg.MinHoldAfterSwitchMs = 0
	state := NewState(cfg)
	state.MainAudioID = "alice"
	now := time.Now()

	streams := []protocol.StreamMetrics{
		metric("alice", 0.5, true),
		metric("bob", 0.7, true),
	}
	sel := Select(state, cfg, streams, now)
	if sel.MixerState != protocol.MixerHold {
		t.Fatalf("expected HOLD before timer, got %v", sel.MixerState)
	}

	sel = Select(state, cfg, streams, now.Add(500*time.Millisecond))
	if sel.ActiveAudioID != "bob" {
		t.Fatalf("expected switch to bob, got %v", sel.ActiveAudioID)
	}
}

func TestEmergencySwitch(t *testing.T) {
	cfg := defaultCfg()
	cfg.EmergencyScoreRatio = 2
	cfg.AudioHoldMs = 1000
	cfg.MinHoldAfterSwitchMs = 0
	state := NewState(cfg)
	state.MainAudioID = "alice"
	now := time.Now()

	streams := []protocol.StreamMetrics{
		metric("alice", 0.1, true),
		metric("bob", 0.9, true),
	}
	sel := Select(state, cfg, streams, now)
	if sel.ActiveAudioID != "bob" || sel.MixerState != protocol.MixerSwitch {
		t.Fatalf("emergency switch failed: audio=%v state=%v", sel.ActiveAudioID, sel.MixerState)
	}
}

func TestHostWinsOnScoreNotVAD(t *testing.T) {
	cfg := defaultCfg()
	cfg.HostPriority = 1
	state := NewState(cfg)
	state.MainAudioID = protocol.HostStreamID

	streams := []protocol.StreamMetrics{
		hostMetric(0.6, false),
		metric("alice", 0.55, true),
	}
	sel := Select(state, cfg, streams, time.Now())
	if sel.ActiveAudioID != protocol.HostStreamID {
		t.Fatalf("host should win on score, got %v", sel.ActiveAudioID)
	}
}

func TestCrossfadeAdvances(t *testing.T) {
	cfg := defaultCfg()
	cfg.CrossfadeMs = 100
	cfg.MinHoldAfterSwitchMs = 0
	cfg.AudioHoldMs = 0
	cfg.SwitchMargin = 0
	state := NewState(cfg)
	state.MainAudioID = "alice"
	state.crossfade = &protocol.CrossfadeState{FromID: "alice", ToID: "bob", T: 0}

	streams := []protocol.StreamMetrics{
		metric("bob", 0.9, true),
	}
	sel := Select(state, cfg, streams, time.Now())
	if sel.Crossfade == nil || sel.Crossfade.T <= 0 {
		t.Fatalf("crossfade should advance, got %v", sel.Crossfade)
	}
}
