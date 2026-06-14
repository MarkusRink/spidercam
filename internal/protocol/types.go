package protocol

type LoopDelayEstimate struct {
	Ms            *int    `json:"ms"`
	UncertaintyMs float64 `json:"uncertaintyMs"`
	Known         bool    `json:"known"`
}

type ParticipantInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	HasVideo bool   `json:"hasVideo"`
	HasAudio bool   `json:"hasAudio"`
	JoinedAt int64  `json:"joinedAt"`
}

type ScoreComponents struct {
	Level       float64 `json:"level"`
	Snr         float64 `json:"snr"`
	Vad         float64 `json:"vad"`
	Priority    float64 `json:"priority"`
	EchoPenalty float64 `json:"echoPenalty"`
}

type StreamMetrics struct {
	ParticipantID string     `json:"participantId"`
	Name          string     `json:"name"`
	Role          StreamRole `json:"role"`

	RmsDbfs         float64 `json:"rmsDbfs"`
	PeakDbfs        float64 `json:"peakDbfs"`
	SpeechLevelDbfs float64 `json:"speechLevelDbfs"`
	NoiseFloorDbfs  float64 `json:"noiseFloorDbfs"`
	SnrDb           float64 `json:"snrDb"`

	Vad             bool            `json:"vad"`
	VadHangoverMs   int             `json:"vadHangoverMs"`
	Score           float64         `json:"score"`
	ScoreSmooth     float64         `json:"scoreSmooth"`
	ScoreComponents ScoreComponents `json:"scoreComponents"`
	Rank            int             `json:"rank"`

	GateGainDb       float64 `json:"gateGainDb"`
	DuckingGainDb    float64 `json:"duckingGainDb"`
	CalibrationGain  float64 `json:"calibrationGain"`
	CalibrationPhase string  `json:"calibrationPhase"`

	JitterBufferFrames int     `json:"jitterBufferFrames"`
	DelayOffsetMs      float64 `json:"delayOffsetMs"`
	IsMainTalker       bool    `json:"isMainTalker"`

	VideoActive bool `json:"videoActive"`
	AudioActive bool `json:"audioActive"`

	RttMs           *float64 `json:"rttMs"`
	PacketLoss      *float64 `json:"packetLoss"`
	JitterMs        *float64 `json:"jitterMs"`
	BitrateKbps     *float64 `json:"bitrateKbps"`
	FramesPerSecond *float64 `json:"framesPerSecond"`

	LastUpdated int64             `json:"lastUpdated"`
	LoopDelay   LoopDelayEstimate `json:"loopDelay"`

	AecEnabled     bool    `json:"aecEnabled"`
	DenoiseEnabled bool    `json:"denoiseEnabled"`
	AecUs          float64 `json:"aecUs"`
	DenoiseUs      float64 `json:"denoiseUs"`
}

type StreamProcessingFlags struct {
	AecEnabled     bool `json:"aecEnabled"`
	DenoiseEnabled bool `json:"denoiseEnabled"`
}

type ReferenceMetrics struct {
	StreamID string  `json:"streamId"`
	RmsDbfs  float64 `json:"rmsDbfs"`
	PeakDbfs float64 `json:"peakDbfs"`
	Vad      bool    `json:"vad"`
	Active   bool    `json:"active"`
}

type CrossfadeState struct {
	FromID string  `json:"fromId"`
	ToID   string  `json:"toId"`
	T      float64 `json:"t"`
}

type SwitchEvent struct {
	At     int64  `json:"at"`
	FromID string `json:"fromId"`
	ToID   string `json:"toId"`
	Reason string `json:"reason"`
}

type SelectionState struct {
	ActiveVideoID  string          `json:"activeVideoId"`
	ActiveAudioID  string          `json:"activeAudioId"`
	MainTalkerID   string          `json:"mainTalkerId"`
	MixerState     MixerState      `json:"mixerState"`
	HoldRemainingMs int            `json:"holdRemainingMs"`
	Crossfade      *CrossfadeState `json:"crossfade"`
	SwitchEvents   []SwitchEvent   `json:"switchEvents"`
	Reason         string          `json:"reason"`
	Timestamp      int64           `json:"timestamp"`
}

type CaptureState struct {
	MicID       string `json:"micId"`
	MicLabel    string `json:"micLabel"`
	CameraID    string `json:"cameraId"`
	CameraLabel string `json:"cameraLabel"`
	SinkID      string `json:"sinkId"`
	SinkLabel   string `json:"sinkLabel"`
}

type RoomState struct {
	Participants         []ParticipantInfo `json:"participants"`
	Metrics              []StreamMetrics   `json:"metrics"`
	Reference            ReferenceMetrics  `json:"reference"`
	Selection            *SelectionState   `json:"selection"`
	Capture              CaptureState      `json:"capture"`
	OutputHealthy        bool              `json:"outputHealthy"`
	GlobalLatencyMs      *int              `json:"globalLatencyMs"`
	OutLevelDbfs         float64           `json:"outLevelDbfs"`
	OutPeakDbfs          float64           `json:"outPeakDbfs"`
	EnhancementBudgetPct float64           `json:"enhancementBudgetPct"`
	ParticipantURL       string            `json:"participantUrl"`
}

type DeviceInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type CaptureDevices struct {
	Mics    []DeviceInfo `json:"mics"`
	Cameras []DeviceInfo `json:"cameras"`
	Sinks   []DeviceInfo `json:"sinks"`
}

type CaptureSelection struct {
	MicID    string `json:"micId"`
	CameraID string `json:"cameraId"`
	SinkID   string `json:"sinkId"`
}

type SelfMetric struct {
	RmsDbfs          float64           `json:"rmsDbfs"`
	SnrDb            float64           `json:"snrDb"`
	Vad              bool              `json:"vad"`
	CalibrationPhase string            `json:"calibrationPhase"`
	LoopDelay        LoopDelayEstimate `json:"loopDelay"`
}

type ParticipantRoomView struct {
	Participants []ParticipantInfo `json:"participants"`
	Selection    *SelectionState   `json:"selection"`
	SelfMetric   SelfMetric        `json:"selfMetric"`
}
