//go:build !linux

package capture

func listV4L2Cameras() ([]DeviceInfo, error) {
	return nil, nil
}

func openV4L2Camera(deviceID string) (*v4l2Camera, error) {
	return nil, nil
}

type v4l2Camera struct{}

func (c *v4l2Camera) readFrame() ([]byte, int, int, bool) {
	return nil, 0, 0, false
}

func (c *v4l2Camera) close() {}
