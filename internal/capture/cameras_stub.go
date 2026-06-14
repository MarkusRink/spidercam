//go:build !cgo || !linux || !spidercam_native_capture

package capture

func listCameras() ([]DeviceInfo, error) {
	return []DeviceInfo{
		{ID: "mock-cam", Label: "Mock Camera"},
	}, nil
}

func openCamera(deviceID string) (*v4l2Camera, error) {
	return nil, nil
}
