//go:build linux

package capture

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"sort"
	"strings"
	"sync"

	"github.com/blackjack/webcam"
)

type v4l2Camera struct {
	mu     sync.Mutex
	cam    *webcam.Webcam
	format string
	width  uint32
	height uint32
	closed bool
}

func listV4L2Cameras() ([]DeviceInfo, error) {
	paths, err := globVideoDevices()
	if err != nil {
		return nil, err
	}
	out := make([]DeviceInfo, 0, len(paths))
	for _, path := range paths {
		if !canCapture(path) {
			continue
		}
		out = append(out, DeviceInfo{
			ID:    path,
			Label: readCardName(path),
		})
	}
	return out, nil
}

func openV4L2Camera(deviceID string) (*v4l2Camera, error) {
	if deviceID == "" {
		devices, err := listV4L2Cameras()
		if err != nil {
			return nil, err
		}
		if len(devices) == 0 {
			return nil, fmt.Errorf("no v4l2 capture device")
		}
		deviceID = devices[0].ID
	}

	cam, err := webcam.Open(deviceID)
	if err != nil {
		return nil, err
	}

	format, width, height, err := configureCamera(cam)
	if err != nil {
		cam.Close()
		return nil, err
	}

	_ = cam.SetBufferCount(1)
	if err := cam.StartStreaming(); err != nil {
		cam.Close()
		return nil, err
	}

	return &v4l2Camera{
		cam:    cam,
		format: format,
		width:  width,
		height: height,
	}, nil
}

func (c *v4l2Camera) readFrame() ([]byte, int, int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.cam == nil {
		return nil, 0, 0, false
	}

	switch w := c.cam.WaitForFrame(1).(type) {
	case nil:
	case *webcam.Timeout:
		return nil, 0, 0, false
	default:
		_ = w
		return nil, 0, 0, false
	}

	frame, err := c.cam.ReadFrame()
	if err != nil || len(frame) == 0 {
		return nil, 0, 0, false
	}

	rgba, err := frameToRGBA(frame, c.format, c.width, c.height)
	if err != nil {
		return nil, 0, 0, false
	}
	return rgba.Pix, int(c.width), int(c.height), true
}

func (c *v4l2Camera) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	if c.cam != nil {
		_ = c.cam.StopStreaming()
		c.cam.Close()
		c.cam = nil
	}
}

func canCapture(devPath string) bool {
	cam, err := webcam.Open(devPath)
	if err != nil {
		return false
	}
	defer cam.Close()
	_, _, _, err = configureCamera(cam)
	return err == nil
}

func configureCamera(cam *webcam.Webcam) (string, uint32, uint32, error) {
	formats := cam.GetSupportedFormats()
	sizes := [][2]uint32{{1280, 720}, {640, 480}, {800, 600}, {320, 240}}

	type fmtTry struct {
		pix  webcam.PixelFormat
		desc string
	}
	tries := make([]fmtTry, 0, len(formats))
	for pix, desc := range formats {
		tries = append(tries, fmtTry{pix: pix, desc: desc})
	}
	sort.Slice(tries, func(i, j int) bool {
		return formatPriority(tries[i].desc) < formatPriority(tries[j].desc)
	})

	for _, f := range tries {
		for _, size := range sizes {
			gotPix, w, h, err := cam.SetImageFormat(f.pix, size[0], size[1])
			if err == nil {
				return formats[gotPix], w, h, nil
			}
		}
	}
	return "", 0, 0, fmt.Errorf("no capturable format at a supported resolution")
}

func formatPriority(desc string) int {
	switch {
	case strings.Contains(desc, "MJPEG"):
		return 0
	case strings.Contains(desc, "YUYV"):
		return 1
	default:
		return 2
	}
}

func frameToRGBA(frame []byte, format string, width, height uint32) (*image.RGBA, error) {
	switch {
	case strings.Contains(format, "MJPEG"), strings.Contains(format, "JPEG"):
		img, err := jpeg.Decode(bytes.NewReader(frame))
		if err != nil {
			return nil, fmt.Errorf("jpeg decode: %w", err)
		}
		bounds := img.Bounds()
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
		return rgba, nil
	case strings.Contains(format, "YUYV"):
		return yuyvToRGBA(frame, width, height), nil
	default:
		return nil, fmt.Errorf("unsupported pixel format %q", format)
	}
}

func yuyvToRGBA(frame []byte, width, height uint32) *image.RGBA {
	w, h := int(width), int(height)
	yuyv := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio422)
	for i := range yuyv.Cb {
		ii := i * 4
		if ii+3 >= len(frame) {
			break
		}
		yuyv.Y[i*2] = frame[ii]
		yuyv.Y[i*2+1] = frame[ii+2]
		yuyv.Cb[i] = frame[ii+1]
		yuyv.Cr[i] = frame[ii+3]
	}
	b := yuyv.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), yuyv, b.Min, draw.Src)
	return rgba
}

func globVideoDevices() ([]string, error) {
	matches, err := filepathGlob("/dev/video*")
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}
