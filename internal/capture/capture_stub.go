//go:build !cgo || !linux || !spidercam_native_capture

package capture

type nativeHandle struct{}

func (h nativeHandle) active() bool {
	return true
}

func listPipeWireDevices() (mics, sinks []DeviceInfo, err error) {
	return []DeviceInfo{
			{ID: "mock-mic", Label: "Mock Microphone"},
		}, []DeviceInfo{
			{ID: "mock-sink", Label: "Mock Sink"},
		}, nil
}

func openNativeCapture(micID, sinkID string, sampleRate int) (nativeHandle, error) {
	return nativeHandle{}, nil
}

func closeNativeCapture(h nativeHandle) {}

func readNativeMic(h nativeHandle, buf []float32) int {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf)
}

func readNativeMonitor(h nativeHandle, buf []float32) int {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf)
}
