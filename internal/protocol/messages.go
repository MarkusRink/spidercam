package protocol

type WelcomeMsg struct {
	Type     string              `json:"type"`
	ClientID string              `json:"clientId"`
	View     ParticipantRoomView `json:"view"`
}

type ParticipantViewMsg struct {
	Type string              `json:"type"`
	View ParticipantRoomView `json:"view"`
}

type JoinMsg struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	HasVideo bool   `json:"hasVideo"`
	HasAudio bool   `json:"hasAudio"`
	ClientID string `json:"clientId,omitempty"`
}

type LeaveMsg struct {
	Type string `json:"type"`
}

type OfferMsg struct {
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	SDP       string `json:"sdp"`
}

type AnswerMsg struct {
	Type string `json:"type"`
	From string `json:"from"`
	To   string `json:"to"`
	SDP  string `json:"sdp"`
}

type ICECandidateMsg struct {
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Candidate string `json:"candidate"`
}

type ErrorMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type HostStateMsg struct {
	Type  string    `json:"type"`
	State RoomState `json:"state"`
}

type ConfigMsg struct {
	Type   string          `json:"type"`
	Config HostConfigPatch `json:"config"`
}

type ListCaptureDevicesMsg struct {
	Type string `json:"type"`
}

type CaptureDevicesMsg struct {
	Type    string         `json:"type"`
	Devices CaptureDevices `json:"devices"`
}

type SetCaptureDevicesMsg struct {
	Type      string           `json:"type"`
	Selection CaptureSelection `json:"selection"`
}

type CaptureDevicesUpdatedMsg struct {
	Type    string       `json:"type"`
	Capture CaptureState `json:"capture"`
	Error   string       `json:"error,omitempty"`
}

type ParticipantURLMsg struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type CopyParticipantURLMsg struct {
	Type string `json:"type"`
}

type SetStreamProcessingMsg struct {
	Type          string                `json:"type"`
	ParticipantID string                `json:"participantId"`
	Flags         StreamProcessingFlags `json:"flags"`
}

type PreviewStreamInitMsg struct {
	Type   string `json:"type"`
	Codec  string `json:"codec"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	FPS    int    `json:"fps"`
}

type PreviewCutMsg struct {
	Type          string `json:"type"`
	ActiveVideoID string `json:"activeVideoId"`
	Seq           int    `json:"seq"`
}
