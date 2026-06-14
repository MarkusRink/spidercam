package signaling

import (
	"testing"

	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
)

func sampleCaptureDevices() protocol.CaptureDevices {
	return protocol.CaptureDevices{
		Mics:    []protocol.DeviceInfo{{ID: "mic-a", Label: "Mic A"}, {ID: "mic-b", Label: "Mic B"}},
		Cameras: []protocol.DeviceInfo{{ID: "cam-a", Label: "Cam A"}},
		Sinks:   []protocol.DeviceInfo{{ID: "sink-a", Label: "Sink A"}},
	}
}

func TestApplyCaptureSelectionPartialMicOnly(t *testing.T) {
	r := room.New("http://127.0.0.1:1234/")
	devices := sampleCaptureDevices()

	got, err := ApplyCaptureSelection(r, devices, protocol.CaptureSelection{MicID: "mic-b"})
	if err != nil {
		t.Fatalf("apply partial mic selection: %v", err)
	}
	if got.MicID != "mic-b" {
		t.Fatalf("mic id = %q, want mic-b", got.MicID)
	}
	if got.CameraID != "cam-a" {
		t.Fatalf("camera id = %q, want default cam-a", got.CameraID)
	}
	if got.SinkID != "sink-a" {
		t.Fatalf("sink id = %q, want default sink-a", got.SinkID)
	}
}

func TestApplyCaptureSelectionKeepsExistingWhenPatchOmitsField(t *testing.T) {
	r := room.New("http://127.0.0.1:1234/")
	devices := sampleCaptureDevices()
	if _, err := ApplyCaptureSelection(r, devices, protocol.CaptureSelection{
		MicID: "mic-a", CameraID: "cam-a", SinkID: "sink-a",
	}); err != nil {
		t.Fatalf("seed selection: %v", err)
	}

	got, err := ApplyCaptureSelection(r, devices, protocol.CaptureSelection{MicID: "mic-b"})
	if err != nil {
		t.Fatalf("apply partial mic update: %v", err)
	}
	if got.MicID != "mic-b" || got.CameraID != "cam-a" || got.SinkID != "sink-a" {
		t.Fatalf("capture = %+v, want mic-b with existing camera and sink", got)
	}
}

func TestApplyCaptureSelectionRejectsUnknownPatchedID(t *testing.T) {
	r := room.New("http://127.0.0.1:1234/")
	_, err := ApplyCaptureSelection(r, sampleCaptureDevices(), protocol.CaptureSelection{MicID: "missing"})
	if err != errUnknownCaptureDevice {
		t.Fatalf("err = %v, want unknown device id", err)
	}
}

func TestApplyCaptureSelectionFallsBackWhenCurrentDeviceMissing(t *testing.T) {
	r := room.New("http://127.0.0.1:1234/")
	devices := sampleCaptureDevices()
	if _, err := ApplyCaptureSelection(r, devices, protocol.CaptureSelection{
		MicID: "mic-a", CameraID: "cam-a", SinkID: "sink-a",
	}); err != nil {
		t.Fatalf("seed selection: %v", err)
	}

	staleDevices := protocol.CaptureDevices{
		Mics:    []protocol.DeviceInfo{{ID: "mic-a", Label: "Mic A"}, {ID: "mic-b", Label: "Mic B"}},
		Cameras: []protocol.DeviceInfo{{ID: "cam-a", Label: "Cam A"}},
		Sinks:   []protocol.DeviceInfo{{ID: "sink-b", Label: "Sink B"}},
	}
	got, err := ApplyCaptureSelection(r, staleDevices, protocol.CaptureSelection{MicID: "mic-b"})
	if err != nil {
		t.Fatalf("apply partial mic with stale sink list: %v", err)
	}
	if got.MicID != "mic-b" || got.SinkID != "sink-b" {
		t.Fatalf("capture = %+v, want mic-b with sink fallback", got)
	}
}

func TestEnsureDefaultCaptureSelectionFillsEmptyState(t *testing.T) {
	r := room.New("http://127.0.0.1:1234/")
	room.ApplyBootstrapIdle(r)

	got, changed, err := EnsureDefaultCaptureSelection(r, true)
	if err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	if !changed {
		t.Fatal("expected defaults to be applied")
	}
	if got.MicID == "" || got.CameraID == "" || got.SinkID == "" {
		t.Fatalf("capture = %+v, want all ids populated", got)
	}
}
