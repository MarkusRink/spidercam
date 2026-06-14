package room

import (
	"sync"
	"time"

	"github.com/markus/spidercam/internal/protocol"
)

type ConnectedParticipant struct {
	ID       string
	Name     string
	HasVideo bool
	HasAudio bool
	JoinedAt int64
}

type FullStateHook func(state protocol.RoomState) protocol.RoomState

type Room struct {
	mu               sync.RWMutex
	config           protocol.HostConfig
	state            protocol.RoomState
	connected        map[string]ConnectedParticipant
	streamProcessing map[string]protocol.StreamProcessingFlags
}

func New(participantURL string) *Room {
	r := &Room{
		config:           protocol.DefaultHostConfig,
		connected:        make(map[string]ConnectedParticipant),
		streamProcessing: make(map[string]protocol.StreamProcessingFlags),
		state: protocol.RoomState{
			Participants:         nil,
			Metrics:              nil,
			Reference:            protocol.ReferenceMetrics{StreamID: protocol.PlaybackRefStreamID},
			Selection:            nil,
			Capture:              protocol.CaptureState{},
			OutputHealthy:        true,
			EnhancementBudgetPct: 0,
			ParticipantURL:       participantURL,
		},
	}
	r.streamProcessing[protocol.HostStreamID] = protocol.StreamProcessingFlags{}
	return r
}

func (r *Room) Config() protocol.HostConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

func (r *Room) State() protocol.RoomState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneRoomState(r.state)
}

func (r *Room) SetState(state protocol.RoomState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = cloneRoomState(state)
	r.initStreamProcessingLocked()
}

func (r *Room) ReplaceMetrics(metrics []protocol.StreamMetrics) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state.Metrics = cloneMetrics(metrics)
}

func (r *Room) ReplaceSelection(selection protocol.SelectionState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sel := selection
	r.state.Selection = &sel
}

func (r *Room) ActiveVideoID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.state.Selection == nil {
		return ""
	}
	return r.state.Selection.ActiveVideoID
}

func (r *Room) UpdateConfig(patch protocol.HostConfigPatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	merged := protocol.MergeHostConfig(r.config, patch)
	if err := protocol.ValidateHostConfig(merged); err != nil {
		return err
	}
	r.config = merged
	return nil
}

func (r *Room) SetCaptureSelection(micID, cameraID, sinkID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state.Capture.MicID = micID
	r.state.Capture.CameraID = cameraID
	r.state.Capture.SinkID = sinkID
}

func (r *Room) SetStreamProcessing(participantID string, flags protocol.StreamProcessingFlags) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	var metric *protocol.StreamMetrics
	for i := range r.state.Metrics {
		if r.state.Metrics[i].ParticipantID == participantID {
			metric = &r.state.Metrics[i]
			break
		}
	}
	if metric == nil && participantID != protocol.HostStreamID {
		return false
	}
	r.streamProcessing[participantID] = flags
	if metric != nil {
		metric.AecEnabled = flags.AecEnabled
		metric.DenoiseEnabled = flags.DenoiseEnabled
	}
	return true
}

func (r *Room) Join(clientID, name string, hasVideo, hasAudio bool) ConnectedParticipant {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.connected[clientID]; ok {
		existing.Name = name
		existing.HasVideo = hasVideo
		existing.HasAudio = hasAudio
		r.connected[clientID] = existing
		if info := findParticipant(r.state.Participants, clientID); info != nil {
			info.Name = name
			info.HasVideo = hasVideo
			info.HasAudio = hasAudio
		} else {
			r.state.Participants = append(r.state.Participants, protocol.ParticipantInfo{
				ID:       clientID,
				Name:     name,
				HasVideo: hasVideo,
				HasAudio: hasAudio,
				JoinedAt: time.Now().UnixMilli(),
			})
			r.addMetricForLocked(clientID, name, hasVideo, hasAudio)
		}
		return existing
	}

	info := findParticipant(r.state.Participants, clientID)
	if info == nil {
		info = &protocol.ParticipantInfo{
			ID:       clientID,
			Name:     name,
			HasVideo: hasVideo,
			HasAudio: hasAudio,
			JoinedAt: time.Now().UnixMilli(),
		}
		r.state.Participants = append(r.state.Participants, *info)
		r.addMetricForLocked(clientID, name, hasVideo, hasAudio)
	} else {
		info.Name = name
		info.HasVideo = hasVideo
		info.HasAudio = hasAudio
	}

	participant := ConnectedParticipant{
		ID:       clientID,
		Name:     name,
		HasVideo: hasVideo,
		HasAudio: hasAudio,
		JoinedAt: info.JoinedAt,
	}
	r.connected[clientID] = participant
	return participant
}

func (r *Room) Leave(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.connected, clientID)
	r.state.Participants = filterParticipants(r.state.Participants, clientID)
	r.state.Metrics = filterMetrics(r.state.Metrics, clientID)
	delete(r.streamProcessing, clientID)
}

func (r *Room) ViewFor(clientID string) protocol.ParticipantRoomView {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var metric *protocol.StreamMetrics
	for i := range r.state.Metrics {
		if r.state.Metrics[i].ParticipantID == clientID {
			metric = &r.state.Metrics[i]
			break
		}
	}

	selfMetric := protocol.SelfMetric{
		RmsDbfs:          -60,
		SnrDb:            0,
		Vad:              false,
		CalibrationPhase: "idle",
		LoopDelay: protocol.LoopDelayEstimate{
			UncertaintyMs: 0,
			Known:         false,
		},
	}
	if metric != nil {
		selfMetric.RmsDbfs = metric.RmsDbfs
		selfMetric.SnrDb = metric.SnrDb
		selfMetric.Vad = metric.Vad
		selfMetric.CalibrationPhase = metric.CalibrationPhase
		selfMetric.LoopDelay = metric.LoopDelay
	}

	var selection *protocol.SelectionState
	if r.state.Selection != nil {
		sel := *r.state.Selection
		selection = &sel
	}

	return protocol.ParticipantRoomView{
		Participants: cloneParticipants(r.state.Participants),
		Selection:    selection,
		SelfMetric:   selfMetric,
	}
}

func (r *Room) ConnectedIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.connected))
	for id := range r.connected {
		ids = append(ids, id)
	}
	return ids
}

func (r *Room) FullState(hook FullStateHook) protocol.RoomState {
	r.mu.RLock()
	state := cloneRoomState(r.state)
	r.mu.RUnlock()
	if hook != nil {
		return hook(state)
	}
	return state
}

func (r *Room) UpdateState(fn func(*protocol.RoomState)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn(&r.state)
}

func (r *Room) SetCaptureLabels(micLabel, cameraLabel, sinkLabel string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state.Capture.MicLabel = micLabel
	r.state.Capture.CameraLabel = cameraLabel
	r.state.Capture.SinkLabel = sinkLabel
}

func (r *Room) initStreamProcessingLocked() {
	r.streamProcessing[protocol.HostStreamID] = protocol.StreamProcessingFlags{}
	for _, p := range r.state.Participants {
		r.streamProcessing[p.ID] = protocol.StreamProcessingFlags{}
	}
	for _, m := range r.state.Metrics {
		if flags, ok := r.streamProcessing[m.ParticipantID]; ok {
			m.AecEnabled = flags.AecEnabled
			m.DenoiseEnabled = flags.DenoiseEnabled
		} else {
			r.streamProcessing[m.ParticipantID] = protocol.StreamProcessingFlags{
				AecEnabled:     m.AecEnabled,
				DenoiseEnabled: m.DenoiseEnabled,
			}
		}
	}
}

func (r *Room) addMetricForLocked(clientID, name string, hasVideo, hasAudio bool) {
	for _, m := range r.state.Metrics {
		if m.ParticipantID == clientID {
			return
		}
	}

	var template *protocol.StreamMetrics
	for i := range r.state.Metrics {
		if r.state.Metrics[i].Role == protocol.StreamRoleParticipant {
			template = &r.state.Metrics[i]
			break
		}
	}

	now := time.Now().UnixMilli()
	var metric protocol.StreamMetrics
	if template != nil {
		metric = cloneStreamMetric(*template)
		metric.ParticipantID = clientID
		metric.Name = name
		metric.VideoActive = hasVideo
		metric.AudioActive = hasAudio
		metric.LastUpdated = now
	} else {
		metric = defaultParticipantMetric(clientID, name, hasVideo, hasAudio, len(r.state.Participants), now)
	}
	r.state.Metrics = append(r.state.Metrics, metric)
	r.streamProcessing[clientID] = protocol.StreamProcessingFlags{}
}

func findParticipant(participants []protocol.ParticipantInfo, id string) *protocol.ParticipantInfo {
	for i := range participants {
		if participants[i].ID == id {
			return &participants[i]
		}
	}
	return nil
}

func filterParticipants(participants []protocol.ParticipantInfo, id string) []protocol.ParticipantInfo {
	out := participants[:0]
	for _, p := range participants {
		if p.ID != id {
			out = append(out, p)
		}
	}
	return out
}

func filterMetrics(metrics []protocol.StreamMetrics, id string) []protocol.StreamMetrics {
	out := metrics[:0]
	for _, m := range metrics {
		if m.ParticipantID != id {
			out = append(out, m)
		}
	}
	return out
}

func defaultParticipantMetric(id, name string, hasVideo, hasAudio bool, rank int, now int64) protocol.StreamMetrics {
	rtt := 15.0
	packetLoss := 0.1
	jitter := 8.0
	bitrate := 300.0
	fps := 30.0
	return protocol.StreamMetrics{
		ParticipantID:   id,
		Name:            name,
		Role:            protocol.StreamRoleParticipant,
		RmsDbfs:         -40,
		PeakDbfs:        -35,
		SpeechLevelDbfs: -42,
		NoiseFloorDbfs:  -60,
		SnrDb:           12,
		VadHangoverMs:   0,
		Score:           0.1,
		ScoreSmooth:     0.08,
		ScoreComponents: protocol.ScoreComponents{
			Level: 0.05, Snr: 0.1, Vad: 0, Priority: 0, EchoPenalty: 0,
		},
		Rank:                 rank,
		CalibrationGain:      1,
		CalibrationPhase:     "idle",
		JitterBufferFrames:   2,
		VideoActive:          hasVideo,
		AudioActive:          hasAudio,
		RttMs:                &rtt,
		PacketLoss:           &packetLoss,
		JitterMs:             &jitter,
		BitrateKbps:          &bitrate,
		FramesPerSecond:      &fps,
		LastUpdated:          now,
		LoopDelay:            protocol.LoopDelayEstimate{UncertaintyMs: 0, Known: false},
	}
}

func cloneRoomState(s protocol.RoomState) protocol.RoomState {
	out := s
	out.Participants = cloneParticipants(s.Participants)
	out.Metrics = cloneMetrics(s.Metrics)
	if s.Selection != nil {
		sel := cloneSelection(*s.Selection)
		out.Selection = &sel
	}
	return out
}

func cloneParticipants(in []protocol.ParticipantInfo) []protocol.ParticipantInfo {
	if len(in) == 0 {
		return nil
	}
	out := make([]protocol.ParticipantInfo, len(in))
	copy(out, in)
	return out
}

func cloneMetrics(in []protocol.StreamMetrics) []protocol.StreamMetrics {
	if len(in) == 0 {
		return nil
	}
	out := make([]protocol.StreamMetrics, len(in))
	for i, m := range in {
		out[i] = cloneStreamMetric(m)
	}
	return out
}

func cloneStreamMetric(m protocol.StreamMetrics) protocol.StreamMetrics {
	out := m
	if m.RttMs != nil {
		v := *m.RttMs
		out.RttMs = &v
	}
	if m.PacketLoss != nil {
		v := *m.PacketLoss
		out.PacketLoss = &v
	}
	if m.JitterMs != nil {
		v := *m.JitterMs
		out.JitterMs = &v
	}
	if m.BitrateKbps != nil {
		v := *m.BitrateKbps
		out.BitrateKbps = &v
	}
	if m.FramesPerSecond != nil {
		v := *m.FramesPerSecond
		out.FramesPerSecond = &v
	}
	if m.LoopDelay.Ms != nil {
		v := *m.LoopDelay.Ms
		out.LoopDelay.Ms = &v
	}
	return out
}

func cloneSelection(s protocol.SelectionState) protocol.SelectionState {
	out := s
	if s.Crossfade != nil {
		cf := *s.Crossfade
		out.Crossfade = &cf
	}
	if s.SwitchEvents != nil {
		out.SwitchEvents = make([]protocol.SwitchEvent, len(s.SwitchEvents))
		copy(out.SwitchEvents, s.SwitchEvents)
	}
	return out
}
