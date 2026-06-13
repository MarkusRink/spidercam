# Host configuration

**Target:** `internal/protocol/config.go`, settings panel on `:1235`

Config and capture selection live in **daemon RAM for the session only** — loaded from `DefaultHostConfig` and env/bootstrap defaults at startup; reset on restart (D15). No `host-config.json` or `devices.json` writes.

## Defaults

```go
var DefaultScoreWeights = ScoreWeights{
	Level: 0.35, Snr: 0.35, Vad: 0.25, Priority: 0.2, EchoPenalty: 0.35,
}

var DefaultHostConfig = HostConfig{
	DefaultVideoID: HostStreamID,
	DefaultAudioID: HostStreamID,

	SilenceScoreThreshold: 0.15,
	VideoHoldMs:           300,
	AudioHoldMs:           400,
	MinHoldAfterSwitchMs:  600,
	CrossfadeMs:           100,
	SwitchMargin:          1.0,
	EmergencyScoreRatio:   3.0,
	ScoreSmoothingAlpha:   0.1,
	ScoreWeights:          DefaultScoreWeights,

	HostPriority: 1.0,

	TargetSpeechDbfs:      -20,
	CalibrationGainClampDb: [-12, 18],

	VadSnrOnDb:     7,
	VadSnrOffDb:    3,
	VadHangoverMs:  150,

	GateAttenuationDb: 12,

	ReferenceVadOnDbfs:  -35,
	ReferenceVadOffDbfs: -45,
	ReferenceDuckDb:     -12,

	ReferenceDelayMs: 0,

	LoopDelayScaleMaxMs:  100,
	LoopDelayWindowMs:    500,
	LoopDelayLagSearchMs: 300,
	LoopDelayAnalysisMs:  250,
	LoopDelayPublishMs:   3000,
	LoopDelayMinSamples:  3,
	LoopDelayMinPeak:     0.25,
	LoopDelayStaleMs:     300_000,
}
```

## Settings panel fields

All controls in the always-visible right column ([ui/host-console.md](../ui/host-console.md)). Changes apply immediately via partial `config` (debounced ~150 ms on sliders).

### Devices (WS `set-capture-devices`, not in `HostConfig`)

| UI label        | Key        | Source                           |
| --------------- | ---------- | -------------------------------- |
| Microphone      | `micId`    | PipeWire sources (C enum)        |
| Webcam          | `cameraId` | v4l2                             |
| Playback output | `sinkId`   | PipeWire sinks → monitor for ref |

### Mixer

| UI label      | Key               | Range               |
| ------------- | ----------------- | ------------------- |
| Hold time     | `audioHoldMs`     | 200–800             |
| Crossfade     | `crossfadeMs`     | 50–200 (audio only) |
| Ducking       | `referenceDuckDb` | 0 … −12 dB          |
| Switch margin | `switchMargin`    | 0.5–2.0             |

Per-stream **AEC** and **RNNoise** are not in `HostConfig` — toggled on stream cards via `set-stream-processing` ([domain/messages.md](./messages.md)); session RAM only.

**Ducking:** `referenceDuckDb == 0` → no attenuation when reference active. Negative values apply that dB gain on participant mics while `reference.active`. Replaces separate enable toggle (D12).

Hint under ducking: “Attenuates room mics while remote Teams speech is active. Set to 0 dB to disable.”

### Score weights

| UI label     | Key                        | Range |
| ------------ | -------------------------- | ----- |
| Level        | `scoreWeights.level`       | 0–1   |
| SNR          | `scoreWeights.snr`         | 0–1   |
| VAD          | `scoreWeights.vad`         | 0–1   |
| Priority     | `scoreWeights.priority`    | 0–1   |
| Echo penalty | `scoreWeights.echoPenalty` | 0–1   |

No preset buttons.

## Bootstrap only

Env vars (`SPIDERCAM_MIC`, `SPIDERCAM_CAMERA`, `SPIDERCAM_PLAYBACK_SINK`) seed initial device selection when the UI has not yet sent `set-capture-devices`. Not written back to disk.
