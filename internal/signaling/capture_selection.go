package signaling

import (
	"errors"

	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
)

var errUnknownCaptureDevice = errors.New("unknown device id")

func ApplyCaptureSelection(r *room.Room, devices protocol.CaptureDevices, patch protocol.CaptureSelection) (protocol.CaptureState, error) {
	current := r.State().Capture

	mic, ok := resolveCaptureDevice(devices.Mics, patch.MicID, current.MicID, false)
	if !ok {
		return current, errUnknownCaptureDevice
	}
	camera, ok := resolveCaptureDevice(devices.Cameras, patch.CameraID, current.CameraID, true)
	if !ok {
		return current, errUnknownCaptureDevice
	}
	sink, ok := resolveCaptureDevice(devices.Sinks, patch.SinkID, current.SinkID, false)
	if !ok {
		return current, errUnknownCaptureDevice
	}

	r.SetCaptureSelection(mic.ID, camera.ID, sink.ID)
	r.SetCaptureLabels(mic.Label, camera.Label, sink.Label)
	return r.State().Capture, nil
}

func EnsureDefaultCapture(r *room.Room, devices protocol.CaptureDevices) (protocol.CaptureState, bool, error) {
	current := r.State().Capture
	if current.MicID != "" && current.CameraID != "" && current.SinkID != "" {
		return current, false, nil
	}
	next, err := ApplyCaptureSelection(r, devices, protocol.CaptureSelection{})
	if err != nil {
		return current, false, err
	}
	return next, true, nil
}

func EnsureDefaultCaptureSelection(r *room.Room, useFixtures bool) (protocol.CaptureState, bool, error) {
	current := r.State().Capture
	if current.MicID != "" && current.CameraID != "" && current.SinkID != "" {
		return current, false, nil
	}
	devices, err := ListCaptureDevices(useFixtures)
	if err != nil {
		return current, false, err
	}
	return EnsureDefaultCapture(r, devices)
}

func resolveCaptureDevice(devs []protocol.DeviceInfo, patchID, currentID string, keepStaleCurrent bool) (protocol.DeviceInfo, bool) {
	if patchID != "" {
		return deviceByID(devs, patchID)
	}
	if currentID != "" {
		if dev, ok := deviceByID(devs, currentID); ok {
			return dev, true
		}
		if keepStaleCurrent {
			return protocol.DeviceInfo{ID: currentID, Label: currentID}, true
		}
	}
	return deviceByID(devs, firstDeviceID(devs))
}

func firstDeviceID(devs []protocol.DeviceInfo) string {
	if len(devs) == 0 {
		return ""
	}
	return devs[0].ID
}

func deviceByID(devs []protocol.DeviceInfo, id string) (protocol.DeviceInfo, bool) {
	if id == "" {
		return protocol.DeviceInfo{}, false
	}
	for _, d := range devs {
		if d.ID == id {
			return d, true
		}
	}
	return protocol.DeviceInfo{}, false
}
