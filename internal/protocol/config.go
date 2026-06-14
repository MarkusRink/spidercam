package protocol

type ScoreWeights struct {
	Level       float64 `json:"level"`
	Snr         float64 `json:"snr"`
	Vad         float64 `json:"vad"`
	Priority    float64 `json:"priority"`
	EchoPenalty float64 `json:"echoPenalty"`
}

type HostConfig struct {
	DefaultVideoID          string       `json:"defaultVideoId"`
	DefaultAudioID          string       `json:"defaultAudioId"`
	SilenceScoreThreshold   float64      `json:"silenceScoreThreshold"`
	VideoHoldMs             int          `json:"videoHoldMs"`
	AudioHoldMs             int          `json:"audioHoldMs"`
	MinHoldAfterSwitchMs    int          `json:"minHoldAfterSwitchMs"`
	CrossfadeMs             int          `json:"crossfadeMs"`
	SwitchMargin            float64      `json:"switchMargin"`
	EmergencyScoreRatio     float64      `json:"emergencyScoreRatio"`
	ScoreSmoothingAlpha     float64      `json:"scoreSmoothingAlpha"`
	ScoreWeights            ScoreWeights `json:"scoreWeights"`
	HostPriority            float64      `json:"hostPriority"`
	TargetSpeechDbfs        float64      `json:"targetSpeechDbfs"`
	CalibrationGainClampDb  [2]float64   `json:"calibrationGainClampDb"`
	VadSnrOnDb              float64      `json:"vadSnrOnDb"`
	VadSnrOffDb             float64      `json:"vadSnrOffDb"`
	VadHangoverMs           int          `json:"vadHangoverMs"`
	GateAttenuationDb       float64      `json:"gateAttenuationDb"`
	ReferenceVadOnDbfs      float64      `json:"referenceVadOnDbfs"`
	ReferenceVadOffDbfs     float64      `json:"referenceVadOffDbfs"`
	ReferenceDuckDb         float64      `json:"referenceDuckDb"`
	ReferenceDelayMs        int          `json:"referenceDelayMs"`
	LoopDelayScaleMaxMs     int          `json:"loopDelayScaleMaxMs"`
	LoopDelayWindowMs       int          `json:"loopDelayWindowMs"`
	LoopDelayLagSearchMs    int          `json:"loopDelayLagSearchMs"`
	LoopDelayAnalysisMs     int          `json:"loopDelayAnalysisMs"`
	LoopDelayPublishMs      int          `json:"loopDelayPublishMs"`
	LoopDelayMinSamples     int          `json:"loopDelayMinSamples"`
	LoopDelayMinPeak        float64      `json:"loopDelayMinPeak"`
	LoopDelayStaleMs        int          `json:"loopDelayStaleMs"`
}

var DefaultScoreWeights = ScoreWeights{
	Level:       0.35,
	Snr:         0.35,
	Vad:         0.25,
	Priority:    0.2,
	EchoPenalty: 0.35,
}

var DefaultHostConfig = HostConfig{
	DefaultVideoID:         HostStreamID,
	DefaultAudioID:         HostStreamID,
	SilenceScoreThreshold:  0.15,
	VideoHoldMs:            300,
	AudioHoldMs:            400,
	MinHoldAfterSwitchMs:   600,
	CrossfadeMs:            100,
	SwitchMargin:           1.0,
	EmergencyScoreRatio:    3.0,
	ScoreSmoothingAlpha:    0.1,
	ScoreWeights:           DefaultScoreWeights,
	HostPriority:           1.0,
	TargetSpeechDbfs:       -20,
	CalibrationGainClampDb: [2]float64{-12, 18},
	VadSnrOnDb:             7,
	VadSnrOffDb:            3,
	VadHangoverMs:          150,
	GateAttenuationDb:      12,
	ReferenceVadOnDbfs:     -35,
	ReferenceVadOffDbfs:    -45,
	ReferenceDuckDb:        -12,
	ReferenceDelayMs:       0,
	LoopDelayScaleMaxMs:    100,
	LoopDelayWindowMs:      500,
	LoopDelayLagSearchMs:   300,
	LoopDelayAnalysisMs:    250,
	LoopDelayPublishMs:     3000,
	LoopDelayMinSamples:    3,
	LoopDelayMinPeak:       0.25,
	LoopDelayStaleMs:       300_000,
}

type ScoreWeightsPatch struct {
	Level       *float64 `json:"level,omitempty"`
	Snr         *float64 `json:"snr,omitempty"`
	Vad         *float64 `json:"vad,omitempty"`
	Priority    *float64 `json:"priority,omitempty"`
	EchoPenalty *float64 `json:"echoPenalty,omitempty"`
}

type HostConfigPatch struct {
	DefaultVideoID         *string            `json:"defaultVideoId,omitempty"`
	DefaultAudioID         *string            `json:"defaultAudioId,omitempty"`
	SilenceScoreThreshold  *float64           `json:"silenceScoreThreshold,omitempty"`
	VideoHoldMs            *int               `json:"videoHoldMs,omitempty"`
	AudioHoldMs            *int               `json:"audioHoldMs,omitempty"`
	MinHoldAfterSwitchMs   *int               `json:"minHoldAfterSwitchMs,omitempty"`
	CrossfadeMs            *int               `json:"crossfadeMs,omitempty"`
	SwitchMargin           *float64           `json:"switchMargin,omitempty"`
	EmergencyScoreRatio    *float64           `json:"emergencyScoreRatio,omitempty"`
	ScoreSmoothingAlpha    *float64           `json:"scoreSmoothingAlpha,omitempty"`
	ScoreWeights           *ScoreWeightsPatch `json:"scoreWeights,omitempty"`
	HostPriority           *float64           `json:"hostPriority,omitempty"`
	TargetSpeechDbfs       *float64           `json:"targetSpeechDbfs,omitempty"`
	CalibrationGainClampDb *[2]float64        `json:"calibrationGainClampDb,omitempty"`
	VadSnrOnDb             *float64           `json:"vadSnrOnDb,omitempty"`
	VadSnrOffDb            *float64           `json:"vadSnrOffDb,omitempty"`
	VadHangoverMs          *int               `json:"vadHangoverMs,omitempty"`
	GateAttenuationDb      *float64           `json:"gateAttenuationDb,omitempty"`
	ReferenceVadOnDbfs     *float64           `json:"referenceVadOnDbfs,omitempty"`
	ReferenceVadOffDbfs    *float64           `json:"referenceVadOffDbfs,omitempty"`
	ReferenceDuckDb        *float64           `json:"referenceDuckDb,omitempty"`
	ReferenceDelayMs       *int               `json:"referenceDelayMs,omitempty"`
	LoopDelayScaleMaxMs    *int               `json:"loopDelayScaleMaxMs,omitempty"`
	LoopDelayWindowMs      *int               `json:"loopDelayWindowMs,omitempty"`
	LoopDelayLagSearchMs   *int               `json:"loopDelayLagSearchMs,omitempty"`
	LoopDelayAnalysisMs    *int               `json:"loopDelayAnalysisMs,omitempty"`
	LoopDelayPublishMs     *int               `json:"loopDelayPublishMs,omitempty"`
	LoopDelayMinSamples    *int               `json:"loopDelayMinSamples,omitempty"`
	LoopDelayMinPeak       *float64           `json:"loopDelayMinPeak,omitempty"`
	LoopDelayStaleMs       *int               `json:"loopDelayStaleMs,omitempty"`
}

func MergeHostConfig(base HostConfig, patch HostConfigPatch) HostConfig {
	merged := base
	if patch.DefaultVideoID != nil {
		merged.DefaultVideoID = *patch.DefaultVideoID
	}
	if patch.DefaultAudioID != nil {
		merged.DefaultAudioID = *patch.DefaultAudioID
	}
	if patch.SilenceScoreThreshold != nil {
		merged.SilenceScoreThreshold = *patch.SilenceScoreThreshold
	}
	if patch.VideoHoldMs != nil {
		merged.VideoHoldMs = *patch.VideoHoldMs
	}
	if patch.AudioHoldMs != nil {
		merged.AudioHoldMs = *patch.AudioHoldMs
	}
	if patch.MinHoldAfterSwitchMs != nil {
		merged.MinHoldAfterSwitchMs = *patch.MinHoldAfterSwitchMs
	}
	if patch.CrossfadeMs != nil {
		merged.CrossfadeMs = *patch.CrossfadeMs
	}
	if patch.SwitchMargin != nil {
		merged.SwitchMargin = *patch.SwitchMargin
	}
	if patch.EmergencyScoreRatio != nil {
		merged.EmergencyScoreRatio = *patch.EmergencyScoreRatio
	}
	if patch.ScoreSmoothingAlpha != nil {
		merged.ScoreSmoothingAlpha = *patch.ScoreSmoothingAlpha
	}
	if patch.ScoreWeights != nil {
		merged.ScoreWeights = mergeScoreWeights(merged.ScoreWeights, *patch.ScoreWeights)
	}
	if patch.HostPriority != nil {
		merged.HostPriority = *patch.HostPriority
	}
	if patch.TargetSpeechDbfs != nil {
		merged.TargetSpeechDbfs = *patch.TargetSpeechDbfs
	}
	if patch.CalibrationGainClampDb != nil {
		merged.CalibrationGainClampDb = *patch.CalibrationGainClampDb
	}
	if patch.VadSnrOnDb != nil {
		merged.VadSnrOnDb = *patch.VadSnrOnDb
	}
	if patch.VadSnrOffDb != nil {
		merged.VadSnrOffDb = *patch.VadSnrOffDb
	}
	if patch.VadHangoverMs != nil {
		merged.VadHangoverMs = *patch.VadHangoverMs
	}
	if patch.GateAttenuationDb != nil {
		merged.GateAttenuationDb = *patch.GateAttenuationDb
	}
	if patch.ReferenceVadOnDbfs != nil {
		merged.ReferenceVadOnDbfs = *patch.ReferenceVadOnDbfs
	}
	if patch.ReferenceVadOffDbfs != nil {
		merged.ReferenceVadOffDbfs = *patch.ReferenceVadOffDbfs
	}
	if patch.ReferenceDuckDb != nil {
		merged.ReferenceDuckDb = *patch.ReferenceDuckDb
	}
	if patch.ReferenceDelayMs != nil {
		merged.ReferenceDelayMs = *patch.ReferenceDelayMs
	}
	if patch.LoopDelayScaleMaxMs != nil {
		merged.LoopDelayScaleMaxMs = *patch.LoopDelayScaleMaxMs
	}
	if patch.LoopDelayWindowMs != nil {
		merged.LoopDelayWindowMs = *patch.LoopDelayWindowMs
	}
	if patch.LoopDelayLagSearchMs != nil {
		merged.LoopDelayLagSearchMs = *patch.LoopDelayLagSearchMs
	}
	if patch.LoopDelayAnalysisMs != nil {
		merged.LoopDelayAnalysisMs = *patch.LoopDelayAnalysisMs
	}
	if patch.LoopDelayPublishMs != nil {
		merged.LoopDelayPublishMs = *patch.LoopDelayPublishMs
	}
	if patch.LoopDelayMinSamples != nil {
		merged.LoopDelayMinSamples = *patch.LoopDelayMinSamples
	}
	if patch.LoopDelayMinPeak != nil {
		merged.LoopDelayMinPeak = *patch.LoopDelayMinPeak
	}
	if patch.LoopDelayStaleMs != nil {
		merged.LoopDelayStaleMs = *patch.LoopDelayStaleMs
	}
	return merged
}

func mergeScoreWeights(base ScoreWeights, patch ScoreWeightsPatch) ScoreWeights {
	merged := base
	if patch.Level != nil {
		merged.Level = *patch.Level
	}
	if patch.Snr != nil {
		merged.Snr = *patch.Snr
	}
	if patch.Vad != nil {
		merged.Vad = *patch.Vad
	}
	if patch.Priority != nil {
		merged.Priority = *patch.Priority
	}
	if patch.EchoPenalty != nil {
		merged.EchoPenalty = *patch.EchoPenalty
	}
	return merged
}

func ValidateHostConfig(cfg HostConfig) error {
	if cfg.CrossfadeMs < 0 || cfg.CrossfadeMs > 5000 {
		return errCrossfadeMsOutOfRange
	}
	if cfg.AudioHoldMs < 0 || cfg.AudioHoldMs > 10_000 {
		return errAudioHoldMsOutOfRange
	}
	return nil
}
