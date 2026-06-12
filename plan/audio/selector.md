# Selector

**Target:** `internal/selector/`

Score-based hysteresis — port to Go. **Exclude** `playback-ref` from candidates. **No host VAD override** (D8): host mic competes on score like any stream.

```go
package selector

type State struct {
	MainAudioID string
	MainVideoID string
	// ... hold timers, crossfade state
}

func Select(state *State, cfg protocol.HostConfig, streams []protocol.StreamMetrics, now time.Time) protocol.SelectionState
```

Filter before rank:

```go
candidates := filter(streams, func(s protocol.StreamMetrics) bool {
	return s.ParticipantID != protocol.PlaybackRefStreamID && s.AudioActive
})
```

## Algorithm

1. **Silence** — if no candidate above `silenceScoreThreshold`, `mixerState: SILENCE`, near-silence output.
2. **Rank** — sort by `scoreSmooth`; host may lead via `HostPriority` in score components, not VAD.
3. **Hold** — incumbent keeps main talker until challenger exceeds margin (`switchMargin`) for `audioHoldMs`.
4. **Emergency switch** — immediate switch if challenger `scoreSmooth ≥ emergencyScoreRatio × incumbent`.
5. **Video** — `activeVideoId` follows talker with `videoHoldMs` lag; **hard cut** in compositor (no video crossfade).
6. **Crossfade** — equal-power blend over `crossfadeMs` on **audio switch only**; sets `mixerState: SWITCH` for timeline UI.

No branch on host VAD. No force-host. See [DECISIONS.resolved.md](../DECISIONS.resolved.md#d8).

## Tests

`selector_test.go`: silence, margin+hold, emergency, ref excluded, host wins only on score not VAD flag, crossfade advances audio only.
