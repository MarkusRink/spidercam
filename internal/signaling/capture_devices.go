package signaling

import (
	"github.com/markus/spidercam/internal/capture"
	"github.com/markus/spidercam/internal/fixtures"
	"github.com/markus/spidercam/internal/protocol"
)

func ListCaptureDevices(useFixtures bool) (protocol.CaptureDevices, error) {
	if useFixtures {
		return fixtures.LoadCaptureDevices()
	}
	devs, err := capture.ListDevices()
	if err != nil {
		return protocol.CaptureDevices{}, err
	}
	return protocol.CaptureDevices{
		Mics:    devs.Mics,
		Cameras: devs.Cameras,
		Sinks:   devs.Sinks,
	}, nil
}
