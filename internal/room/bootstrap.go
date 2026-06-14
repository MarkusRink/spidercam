package room

import (
	"time"

	"github.com/markus/spidercam/internal/capture"
	"github.com/markus/spidercam/internal/protocol"
)

func ApplyBootstrapIdle(r *Room) {
	now := time.Now().UnixMilli()
	envSel := capture.SelectionFromEnv()

	r.SetState(protocol.RoomState{
		Participants: nil,
		Metrics:      []protocol.StreamMetrics{defaultHostMetric(now)},
		Reference: protocol.ReferenceMetrics{
			StreamID: protocol.PlaybackRefStreamID,
			RmsDbfs:  -60,
			PeakDbfs: -60,
			Vad:      false,
			Active:   false,
		},
		Selection: &protocol.SelectionState{
			ActiveVideoID:   protocol.HostStreamID,
			ActiveAudioID:   protocol.HostStreamID,
			MainTalkerID:    protocol.HostStreamID,
			MixerState:      protocol.MixerSilence,
			HoldRemainingMs: 0,
			Crossfade:       nil,
			SwitchEvents:    nil,
			Reason:          "silence",
			Timestamp:       now,
		},
		Capture: protocol.CaptureState{
			MicID:    envSel.MicID,
			CameraID: envSel.CameraID,
			SinkID:   envSel.SinkID,
		},
		OutputHealthy:        true,
		GlobalLatencyMs:      nil,
		OutLevelDbfs:         -60,
		OutPeakDbfs:          -60,
		EnhancementBudgetPct: 0,
		ParticipantURL:       r.State().ParticipantURL,
	})
}

func defaultHostMetric(now int64) protocol.StreamMetrics {
	return protocol.StreamMetrics{
		ParticipantID:   protocol.HostStreamID,
		Name:            "Host",
		Role:            protocol.StreamRoleHost,
		RmsDbfs:         -48.2,
		PeakDbfs:        -44.0,
		SpeechLevelDbfs: -50.0,
		NoiseFloorDbfs:  -62.0,
		SnrDb:           12.0,
		Vad:             false,
		VadHangoverMs:   0,
		Score:           0.08,
		ScoreSmooth:     0.06,
		ScoreComponents: protocol.ScoreComponents{
			Level: 0.05, Snr: 0.1, Vad: 0, Priority: 0.2, EchoPenalty: 0,
		},
		Rank:               1,
		GateGainDb:         0,
		DuckingGainDb:      0,
		CalibrationGain:    1.0,
		CalibrationPhase:   "idle",
		JitterBufferFrames: 0,
		DelayOffsetMs:      0,
		IsMainTalker:       false,
		VideoActive:        true,
		AudioActive:        true,
		LastUpdated:        now,
		LoopDelay:          protocol.LoopDelayEstimate{UncertaintyMs: 0, Known: false},
	}
}
