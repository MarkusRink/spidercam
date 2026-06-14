package fixtures

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/markus/spidercam/internal/protocol"
)

var (
	repoRootOnce sync.Once
	repoRootPath string
)

func repoRoot() string {
	repoRootOnce.Do(func() {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return
		}
		repoRootPath = filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	})
	return repoRootPath
}

func readFile(rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(repoRoot(), rel))
}

type hostStateFixture struct {
	State protocol.RoomState `json:"state"`
}

func LoadRoutingState() (protocol.RoomState, error) {
	data, err := readFile("web/test-fixtures/host-state/routing.json")
	if err != nil {
		return protocol.RoomState{}, err
	}
	var fixture hostStateFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return protocol.RoomState{}, err
	}
	return fixture.State, nil
}

func LoadCaptureDevices() (protocol.CaptureDevices, error) {
	data, err := readFile("web/test-fixtures/capture-devices.json")
	if err != nil {
		return protocol.CaptureDevices{}, err
	}
	var devices protocol.CaptureDevices
	if err := json.Unmarshal(data, &devices); err != nil {
		return protocol.CaptureDevices{}, err
	}
	return devices, nil
}

func LoadPreviewKeyframe() ([]byte, error) {
	return readFile("web/test-fixtures/preview/keyframe.h264")
}
