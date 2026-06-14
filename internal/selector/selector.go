package selector

import (
	"sort"
	"time"

	"github.com/markus/spidercam/internal/protocol"
)

const tickMs = 20

type State struct {
	MainAudioID      string
	MainVideoID      string
	marginHoldStart  time.Time
	minHoldUntil     time.Time
	lastVideoSwitch  time.Time
	pendingVideoID   string
	crossfade        *protocol.CrossfadeState
	switchEvents     []protocol.SwitchEvent
}

func NewState(cfg protocol.HostConfig) *State {
	return &State{
		MainAudioID: cfg.DefaultAudioID,
		MainVideoID: cfg.DefaultVideoID,
	}
}

func Select(state *State, cfg protocol.HostConfig, streams []protocol.StreamMetrics, now time.Time) protocol.SelectionState {
	candidates := filterCandidates(streams)
	ranked := rankByScoreSmooth(candidates)

	active := filterAboveThreshold(ranked, cfg.SilenceScoreThreshold)
	if len(active) == 0 {
		state.crossfade = nil
		return protocol.SelectionState{
			ActiveVideoID:   state.MainVideoID,
			ActiveAudioID:   cfg.DefaultAudioID,
			MainTalkerID:    cfg.DefaultAudioID,
			MixerState:      protocol.MixerSilence,
			HoldRemainingMs: 0,
			Crossfade:       nil,
			SwitchEvents:    cloneEvents(state.switchEvents),
			Reason:          "silence",
			Timestamp:       now.UnixMilli(),
		}
	}

	challenger := &active[0]
	incumbent := findMetric(candidates, state.MainAudioID)
	if incumbent == nil {
		incumbent = findMetric(streams, state.MainAudioID)
	}
	if incumbent == nil {
		incumbent = challenger
	}

	mixerState := protocol.MixerLocked
	reason := "locked"
	audioID := state.MainAudioID
	if audioID == "" {
		audioID = challenger.ParticipantID
		state.MainAudioID = audioID
	}

	incumbentScore := scoreFor(incumbent)
	challengerScore := scoreFor(challenger)

	if challenger.ParticipantID != state.MainAudioID {
		emergency := challengerScore >= cfg.EmergencyScoreRatio*incumbentScore
		marginMet := challengerScore > incumbentScore+cfg.SwitchMargin

		if emergency {
			audioID, mixerState, reason = switchAudio(state, cfg, challenger.ParticipantID, now, "emergency")
		} else if marginMet {
			if state.marginHoldStart.IsZero() {
				state.marginHoldStart = now
			}
			held := now.Sub(state.marginHoldStart) >= time.Duration(cfg.AudioHoldMs)*time.Millisecond
			canSwitch := now.After(state.minHoldUntil) || state.minHoldUntil.IsZero()
			if held && canSwitch {
				audioID, mixerState, reason = switchAudio(state, cfg, challenger.ParticipantID, now, "score_margin")
			} else {
				mixerState = protocol.MixerHold
				if !marginMet {
					state.marginHoldStart = time.Time{}
				}
				reason = "hold"
			}
		} else {
			state.marginHoldStart = time.Time{}
			mixerState = protocol.MixerHold
			reason = "margin"
		}
	} else {
		state.marginHoldStart = time.Time{}
	}

	videoID := updateVideo(state, cfg, challenger, now)
	crossfade := advanceCrossfade(state, cfg)

	holdRemaining := 0
	if mixerState == protocol.MixerHold && !state.marginHoldStart.IsZero() {
		elapsed := now.Sub(state.marginHoldStart)
		need := time.Duration(cfg.AudioHoldMs) * time.Millisecond
		holdRemaining = int((need - elapsed).Milliseconds())
		if holdRemaining < 0 {
			holdRemaining = 0
		}
	}

	for i := range ranked {
		ranked[i].Rank = i + 1
		ranked[i].IsMainTalker = ranked[i].ParticipantID == audioID
	}

	_ = ranked

	return protocol.SelectionState{
		ActiveVideoID:   videoID,
		ActiveAudioID:   audioID,
		MainTalkerID:    challenger.ParticipantID,
		MixerState:      mixerState,
		HoldRemainingMs: holdRemaining,
		Crossfade:       crossfade,
		SwitchEvents:    cloneEvents(state.switchEvents),
		Reason:          reason,
		Timestamp:       now.UnixMilli(),
	}
}

func filterCandidates(streams []protocol.StreamMetrics) []protocol.StreamMetrics {
	out := make([]protocol.StreamMetrics, 0, len(streams))
	for _, s := range streams {
		if s.ParticipantID == protocol.PlaybackRefStreamID {
			continue
		}
		if !s.AudioActive {
			continue
		}
		out = append(out, s)
	}
	return out
}

func rankByScoreSmooth(streams []protocol.StreamMetrics) []protocol.StreamMetrics {
	out := append([]protocol.StreamMetrics(nil), streams...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScoreSmooth == out[j].ScoreSmooth {
			return out[i].ParticipantID < out[j].ParticipantID
		}
		return out[i].ScoreSmooth > out[j].ScoreSmooth
	})
	return out
}

func filterAboveThreshold(streams []protocol.StreamMetrics, threshold float64) []protocol.StreamMetrics {
	out := make([]protocol.StreamMetrics, 0, len(streams))
	for _, s := range streams {
		if s.ScoreSmooth >= threshold {
			out = append(out, s)
		}
	}
	return out
}

func findMetric(streams []protocol.StreamMetrics, id string) *protocol.StreamMetrics {
	for i := range streams {
		if streams[i].ParticipantID == id {
			return &streams[i]
		}
	}
	return nil
}

func scoreFor(m *protocol.StreamMetrics) float64 {
	if m == nil {
		return 0
	}
	return m.ScoreSmooth
}

func switchAudio(state *State, cfg protocol.HostConfig, toID string, now time.Time, reason string) (string, protocol.MixerState, string) {
	fromID := state.MainAudioID
	if fromID == toID {
		return toID, protocol.MixerLocked, "locked"
	}
	state.switchEvents = append(state.switchEvents, protocol.SwitchEvent{
		At:     now.UnixMilli(),
		FromID: fromID,
		ToID:   toID,
		Reason: reason,
	})
	if len(state.switchEvents) > 50 {
		state.switchEvents = state.switchEvents[len(state.switchEvents)-50:]
	}
	state.MainAudioID = toID
	state.minHoldUntil = now.Add(time.Duration(cfg.MinHoldAfterSwitchMs) * time.Millisecond)
	state.marginHoldStart = time.Time{}
	if cfg.CrossfadeMs > 0 && fromID != "" {
		state.crossfade = &protocol.CrossfadeState{FromID: fromID, ToID: toID, T: 0}
	} else {
		state.crossfade = nil
	}
	return toID, protocol.MixerSwitch, reason
}

func updateVideo(state *State, cfg protocol.HostConfig, best *protocol.StreamMetrics, now time.Time) string {
	target := best.ParticipantID
	if best.ScoreSmooth < cfg.SilenceScoreThreshold {
		target = cfg.DefaultVideoID
	}
	if target == state.MainVideoID {
		state.pendingVideoID = ""
		return state.MainVideoID
	}
	if state.pendingVideoID != target {
		state.pendingVideoID = target
		state.lastVideoSwitch = now
		return state.MainVideoID
	}
	held := now.Sub(state.lastVideoSwitch) >= time.Duration(cfg.VideoHoldMs)*time.Millisecond
	if held {
		state.MainVideoID = target
		state.pendingVideoID = ""
	}
	return state.MainVideoID
}

func advanceCrossfade(state *State, cfg protocol.HostConfig) *protocol.CrossfadeState {
	if state.crossfade == nil {
		return nil
	}
	crossfadeMs := cfg.CrossfadeMs
	if crossfadeMs <= 0 {
		crossfadeMs = 100
	}
	step := float64(tickMs) / float64(crossfadeMs)
	state.crossfade.T += step
	if state.crossfade.T >= 1 {
		done := *state.crossfade
		done.T = 1
		state.crossfade = nil
		return &done
	}
	cf := *state.crossfade
	return &cf
}

func cloneEvents(events []protocol.SwitchEvent) []protocol.SwitchEvent {
	if len(events) == 0 {
		return nil
	}
	out := make([]protocol.SwitchEvent, len(events))
	copy(out, events)
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
