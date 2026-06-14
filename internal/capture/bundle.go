package capture

import (
	"context"
	"os"
	"strconv"
	"sync"

	"github.com/markus/spidercam/internal/protocol"
)

const (
	FrameSamples        = 480
	DefaultSampleRate   = 48000
	maxPipeWireDevices  = 128
	envMic              = "SPIDERCAM_MIC"
	envCamera           = "SPIDERCAM_CAMERA"
	envPlaybackSink     = "SPIDERCAM_PLAYBACK_SINK"
	envSampleRate       = "SPIDERCAM_SAMPLE_RATE"
	envMockCapture      = "SPIDERCAM_MOCK_CAPTURE"
)

type DeviceInfo = protocol.DeviceInfo

type Devices struct {
	Mics    []DeviceInfo `json:"mics"`
	Cameras []DeviceInfo `json:"cameras"`
	Sinks   []DeviceInfo `json:"sinks"`
}

type Selection = protocol.CaptureSelection

type Bundle struct {
	mu         sync.Mutex
	selection  Selection
	labels     deviceLabels
	sampleRate int
	native     nativeHandle
	camera     *v4l2Camera
	closed     bool
}

type deviceLabels struct {
	mic    string
	camera string
	sink   string
}

func SelectionFromEnv() Selection {
	return Selection{
		MicID:    os.Getenv(envMic),
		CameraID: os.Getenv(envCamera),
		SinkID:   os.Getenv(envPlaybackSink),
	}
}

func SampleRateFromEnv() int {
	if v := os.Getenv(envSampleRate); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return DefaultSampleRate
}

func ListDevices() (Devices, error) {
	mics, sinks, err := listPipeWireDevices()
	if err != nil {
		return Devices{}, err
	}
	cameras, err := listCameras()
	if err != nil {
		return Devices{}, err
	}
	return Devices{Mics: mics, Cameras: cameras, Sinks: sinks}, nil
}

func Open(ctx context.Context, sel Selection, sampleRate int) (*Bundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sampleRate <= 0 {
		sampleRate = DefaultSampleRate
	}

	labels, err := resolveLabels(sel)
	if err != nil {
		return nil, err
	}

	native, err := openNativeCapture(sel.MicID, sel.SinkID, sampleRate)
	if err != nil {
		return nil, err
	}

	cam, err := openCamera(sel.CameraID)
	if err != nil && sel.CameraID != "" {
		closeNativeCapture(native)
		return nil, err
	}

	return &Bundle{
		selection:  sel,
		labels:     labels,
		sampleRate: sampleRate,
		native:     native,
		camera:     cam,
	}, nil
}

func (b *Bundle) Reopen(ctx context.Context, sel Selection) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	labels, err := resolveLabels(sel)
	if err != nil {
		return err
	}

	if sel.MicID != b.selection.MicID || sel.SinkID != b.selection.SinkID {
		closeNativeCapture(b.native)
		b.native, err = openNativeCapture(sel.MicID, sel.SinkID, b.sampleRate)
		if err != nil {
			return err
		}
	}

	if sel.CameraID != b.selection.CameraID {
		var newCam *v4l2Camera
		if sel.CameraID != "" {
			newCam, err = openCamera(sel.CameraID)
			if err != nil {
				return err
			}
		}
		if b.camera != nil {
			b.camera.close()
		}
		b.camera = newCam
	}

	b.selection = sel
	b.labels = labels
	return nil
}

func (b *Bundle) Close() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	closeNativeCapture(b.native)
	b.native = nativeHandle{}
	if b.camera != nil {
		b.camera.close()
		b.camera = nil
	}
	return nil
}

func (b *Bundle) ReadMic(buf []float32) int {
	if b == nil || len(buf) == 0 {
		return 0
	}
	b.mu.Lock()
	native := b.native
	closed := b.closed
	b.mu.Unlock()
	if closed || !native.active() {
		return 0
	}
	return readNativeMic(native, buf)
}

func (b *Bundle) ReadMonitor(buf []float32) int {
	if b == nil || len(buf) == 0 {
		return 0
	}
	b.mu.Lock()
	native := b.native
	closed := b.closed
	b.mu.Unlock()
	if closed || !native.active() {
		return 0
	}
	return readNativeMonitor(native, buf)
}

func (b *Bundle) ReadCamera() (rgba []byte, width, height int, ok bool) {
	if b == nil {
		return nil, 0, 0, false
	}
	b.mu.Lock()
	cam := b.camera
	closed := b.closed
	b.mu.Unlock()
	if closed || cam == nil {
		return nil, 0, 0, false
	}
	return cam.readFrame()
}

func (b *Bundle) ActiveSelection() Selection {
	if b == nil {
		return Selection{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.selection
}

func (b *Bundle) ActiveState() protocol.CaptureState {
	if b == nil {
		return protocol.CaptureState{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return protocol.CaptureState{
		MicID:       b.selection.MicID,
		MicLabel:    b.labels.mic,
		CameraID:    b.selection.CameraID,
		CameraLabel: b.labels.camera,
		SinkID:      b.selection.SinkID,
		SinkLabel:   b.labels.sink,
	}
}

func resolveLabels(sel Selection) (deviceLabels, error) {
	devs, err := ListDevices()
	if err != nil {
		return deviceLabels{}, err
	}
	return deviceLabels{
		mic:    labelForID(devs.Mics, sel.MicID),
		camera: labelForID(devs.Cameras, sel.CameraID),
		sink:   labelForID(devs.Sinks, sel.SinkID),
	}, nil
}

func labelForID(devs []DeviceInfo, id string) string {
	if id == "" {
		return ""
	}
	for _, d := range devs {
		if d.ID == id {
			return d.Label
		}
	}
	return id
}
