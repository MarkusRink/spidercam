//go:build cgo && linux && spidercam_native_capture

package capture

func listCameras() ([]DeviceInfo, error) {
	return listV4L2Cameras()
}

func openCamera(deviceID string) (*v4l2Camera, error) {
	return openV4L2Camera(deviceID)
}
