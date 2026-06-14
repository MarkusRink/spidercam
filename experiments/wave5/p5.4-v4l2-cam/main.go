package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/blackjack/webcam"
)

const outputPath = "output/frame.png"

type deviceInfo struct {
	Path string
	Name string
}

func main() {
	deviceFlag := flag.String("device", "", "v4l2 device path (default: first working capture device)")
	flag.Parse()

	devices, err := listDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list devices: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("v4l2 devices:")
	for _, d := range devices {
		fmt.Printf("  %s  %s\n", d.Path, d.Name)
	}

	if len(devices) == 0 {
		fmt.Println("RESULT: skip (no /dev/video* nodes)")
		os.Exit(0)
	}

	target := *deviceFlag
	if target == "" {
		var ok bool
		target, ok = firstWorking(devices)
		if !ok {
			fmt.Println("RESULT: skip (no working capture device)")
			os.Exit(0)
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir output: %v\n", err)
		os.Exit(1)
	}

	format, width, height, err := capturePNG(target, outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture %s: %v\n", target, err)
		fmt.Println("RESULT: fail")
		os.Exit(1)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stat output: %v\n", err)
		fmt.Println("RESULT: fail")
		os.Exit(1)
	}

	size := info.Size()
	fmt.Printf("device: %s\n", target)
	fmt.Printf("format: %s %dx%d\n", format, width, height)
	fmt.Printf("output: %s (%d bytes)\n", outputPath, size)

	if size < 5000 {
		fmt.Println("RESULT: fail (PNG too small — likely blank or corrupt)")
		os.Exit(1)
	}

	fmt.Println("RESULT: pass")
}

func listDevices() ([]deviceInfo, error) {
	matches, err := filepath.Glob("/dev/video*")
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)

	out := make([]deviceInfo, 0, len(matches))
	for _, path := range matches {
		out = append(out, deviceInfo{
			Path: path,
			Name: readCardName(path),
		})
	}
	return out, nil
}

func readCardName(devPath string) string {
	base := filepath.Base(devPath)
	data, err := os.ReadFile(filepath.Join("/sys/class/video4linux", base, "name"))
	if err != nil {
		return "(unknown)"
	}
	return strings.TrimSpace(string(data))
}

func firstWorking(devices []deviceInfo) (string, bool) {
	for _, d := range devices {
		if canCapture(d.Path) {
			return d.Path, true
		}
	}
	return "", false
}

func canCapture(devPath string) bool {
	cam, err := webcam.Open(devPath)
	if err != nil {
		return false
	}
	defer cam.Close()

	formats := cam.GetSupportedFormats()
	return len(formats) > 0
}

func capturePNG(devPath, outPath string) (format string, width, height uint32, err error) {
	cam, err := webcam.Open(devPath)
	if err != nil {
		return "", 0, 0, err
	}
	defer cam.Close()

	format, width, height, err = configureCamera(cam)
	if err != nil {
		return "", 0, 0, err
	}

	_ = cam.SetBufferCount(1)
	if err := cam.StartStreaming(); err != nil {
		return "", 0, 0, err
	}
	defer cam.StopStreaming()

	switch wait := cam.WaitForFrame(5).(type) {
	case nil:
	case *webcam.Timeout:
		return "", 0, 0, wait
	default:
		return "", 0, 0, wait
	}

	frame, err := cam.ReadFrame()
	if err != nil {
		return "", 0, 0, err
	}
	if len(frame) == 0 {
		return "", 0, 0, fmt.Errorf("empty frame")
	}

	rgba, err := frameToRGBA(frame, format, width, height)
	if err != nil {
		return "", 0, 0, err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return "", 0, 0, err
	}
	defer f.Close()

	if err := png.Encode(f, rgba); err != nil {
		return "", 0, 0, err
	}
	return format, width, height, nil
}

func configureCamera(cam *webcam.Webcam) (string, uint32, uint32, error) {
	formats := cam.GetSupportedFormats()
	sizes := [][2]uint32{{640, 480}, {1280, 720}, {320, 240}, {800, 600}}

	for _, want := range []string{"MJPEG", "YUYV"} {
		for pix, desc := range formats {
			if !strings.Contains(desc, want) {
				continue
			}
			for _, size := range sizes {
				gotPix, w, h, err := cam.SetImageFormat(pix, size[0], size[1])
				if err == nil {
					return formats[gotPix], w, h, nil
				}
			}
		}
	}
	return "", 0, 0, fmt.Errorf("no MJPEG or YUYV format at a supported resolution")
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
