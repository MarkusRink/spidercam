package room

import (
	"os"
	"testing"

	"github.com/markus/spidercam/internal/protocol"
)

func TestApplyBootstrapIdle(t *testing.T) {
	r := New("http://192.168.1.10:1234/")
	ApplyBootstrapIdle(r)

	state := r.State()
	if len(state.Participants) != 0 {
		t.Fatalf("participants = %d, want 0", len(state.Participants))
	}
	if len(state.Metrics) != 1 {
		t.Fatalf("metrics = %d, want 1", len(state.Metrics))
	}
	if state.Metrics[0].ParticipantID != protocol.HostStreamID {
		t.Fatalf("metric id = %q, want host", state.Metrics[0].ParticipantID)
	}
	if state.Metrics[0].Role != protocol.StreamRoleHost {
		t.Fatalf("metric role = %q, want host", state.Metrics[0].Role)
	}
	if state.Selection == nil || state.Selection.MixerState != protocol.MixerSilence {
		t.Fatal("expected SILENCE selection")
	}
	if state.GlobalLatencyMs != nil {
		t.Fatal("expected nil global latency")
	}
	if state.OutLevelDbfs != -60 || state.OutPeakDbfs != -60 {
		t.Fatalf("out levels = %v/%v, want -60/-60", state.OutLevelDbfs, state.OutPeakDbfs)
	}
	if state.Reference.RmsDbfs != -60 || state.Reference.Vad {
		t.Fatal("expected silent reference")
	}
	if state.ParticipantURL != "http://192.168.1.10:1234/" {
		t.Fatalf("participant URL = %q", state.ParticipantURL)
	}
}

func TestApplyBootstrapIdleEnvCapture(t *testing.T) {
	t.Setenv("SPIDERCAM_MIC", "pw:source:test")
	t.Setenv("SPIDERCAM_CAMERA", "v4l2:/dev/video0")
	t.Setenv("SPIDERCAM_PLAYBACK_SINK", "pw:sink:test")
	t.Cleanup(func() {
		_ = os.Unsetenv("SPIDERCAM_MIC")
		_ = os.Unsetenv("SPIDERCAM_CAMERA")
		_ = os.Unsetenv("SPIDERCAM_PLAYBACK_SINK")
	})

	r := New("http://127.0.0.1:1234/")
	ApplyBootstrapIdle(r)

	capture := r.State().Capture
	if capture.MicID != "pw:source:test" {
		t.Fatalf("mic id = %q", capture.MicID)
	}
	if capture.CameraID != "v4l2:/dev/video0" {
		t.Fatalf("camera id = %q", capture.CameraID)
	}
	if capture.SinkID != "pw:sink:test" {
		t.Fatalf("sink id = %q", capture.SinkID)
	}
}
