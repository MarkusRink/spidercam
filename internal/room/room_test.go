package room

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/markus/spidercam/internal/protocol"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readFixture(t *testing.T, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read fixture %s: %v", rel, err)
	}
	return data
}

func jsonRoundTrip[T any](t *testing.T, data []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundTrip T
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	return roundTrip
}

func TestHostStateFixtureRoundTrip(t *testing.T) {
	fixtures := []string{
		"web/test-fixtures/host-state/idle.json",
		"web/test-fixtures/host-state/routing.json",
		"web/test-fixtures/host-state/crossfade.json",
	}
	for _, path := range fixtures {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data := readFixture(t, path)
			msg := jsonRoundTrip[protocol.HostStateMsg](t, data)

			room := New(msg.State.ParticipantURL)
			room.SetState(msg.State)
			got := room.FullState(nil)

			if !reflect.DeepEqual(got, msg.State) {
				t.Fatalf("FullState does not match fixture state")
			}

			encoded, err := json.Marshal(protocol.HostStateMsg{Type: "host-state", State: got})
			if err != nil {
				t.Fatalf("marshal host-state: %v", err)
			}
			var again protocol.HostStateMsg
			if err := json.Unmarshal(encoded, &again); err != nil {
				t.Fatalf("re-unmarshal host-state: %v", err)
			}
			if !reflect.DeepEqual(again.State, msg.State) {
				t.Fatalf("host-state JSON round-trip drift")
			}
		})
	}
}

func TestParticipantViewFixture(t *testing.T) {
	data := readFixture(t, "web/test-fixtures/participant-view/connected.json")
	expected := jsonRoundTrip[protocol.ParticipantViewMsg](t, data)

	routing := jsonRoundTrip[protocol.HostStateMsg](t, readFixture(t, "web/test-fixtures/host-state/routing.json"))
	state := routing.State
	if state.Selection != nil {
		state.Selection.SwitchEvents = []protocol.SwitchEvent{}
	}

	room := New(state.ParticipantURL)
	room.SetState(state)
	room.Join("a3f7c2e1-0000-4000-8000-000000000001", "Alice", true, true)

	got := room.ViewFor("a3f7c2e1-0000-4000-8000-000000000001")
	if !reflect.DeepEqual(got, expected.View) {
		t.Fatalf("ViewFor does not match participant-view fixture")
	}

	encoded, err := json.Marshal(protocol.ParticipantViewMsg{Type: "participant-view", View: got})
	if err != nil {
		t.Fatalf("marshal participant-view: %v", err)
	}
	var again protocol.ParticipantViewMsg
	if err := json.Unmarshal(encoded, &again); err != nil {
		t.Fatalf("re-unmarshal participant-view: %v", err)
	}
	if !reflect.DeepEqual(again.View, expected.View) {
		t.Fatalf("participant-view JSON round-trip drift")
	}
}

func TestUpdateConfigMergeScoreWeights(t *testing.T) {
	room := New("http://example.test/")
	patch := protocol.HostConfigPatch{
		ScoreWeights: &protocol.ScoreWeightsPatch{
			Level: ptrFloat(0.5),
		},
	}
	if err := room.UpdateConfig(patch); err != nil {
		t.Fatalf("UpdateConfig: %v", err)
	}
	cfg := room.Config()
	if cfg.ScoreWeights.Level != 0.5 {
		t.Fatalf("expected merged level 0.5, got %v", cfg.ScoreWeights.Level)
	}
	if cfg.ScoreWeights.Snr != protocol.DefaultScoreWeights.Snr {
		t.Fatalf("expected snr preserved from defaults")
	}
}

func TestSetStreamProcessing(t *testing.T) {
	routing := jsonRoundTrip[protocol.HostStateMsg](t, readFixture(t, "web/test-fixtures/host-state/routing.json"))
	room := New(routing.State.ParticipantURL)
	room.SetState(routing.State)

	aliceID := "a3f7c2e1-0000-4000-8000-000000000001"
	if !room.SetStreamProcessing(aliceID, protocol.StreamProcessingFlags{AecEnabled: true, DenoiseEnabled: true}) {
		t.Fatal("SetStreamProcessing returned false")
	}
	state := room.State()
	var aliceMetric *protocol.StreamMetrics
	for i := range state.Metrics {
		if state.Metrics[i].ParticipantID == aliceID {
			aliceMetric = &state.Metrics[i]
			break
		}
	}
	if aliceMetric == nil {
		t.Fatal("alice metric not found")
	}
	if !aliceMetric.AecEnabled || !aliceMetric.DenoiseEnabled {
		t.Fatalf("stream processing flags not applied: %+v", *aliceMetric)
	}
}

func TestJoinLeave(t *testing.T) {
	room := New("http://example.test/")
	room.Join("client-1", "Test", true, false)
	if len(room.ConnectedIDs()) != 1 {
		t.Fatalf("expected one connected client, got %d", len(room.ConnectedIDs()))
	}
	room.Leave("client-1")
	if len(room.ConnectedIDs()) != 0 {
		t.Fatalf("expected no connected clients after leave")
	}
	state := room.State()
	if len(state.Participants) != 0 || len(state.Metrics) != 0 {
		t.Fatalf("expected participant and metrics removed on leave")
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}
