package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	v4l2BufTypeVideoOutput = 2
	v4l2FieldNone          = 1
	v4l2CapVideoOutput     = 0x00000002

	v4l2PixFmtYuyv  = 0x56595559
	v4l2PixFmtRgb24 = 0x33424752

	frameWidth  = 1280
	frameHeight = 720
	frameCount  = 300
	frameFPS    = 30
)

type v4l2Capability struct {
	Driver       [16]byte
	Card         [32]byte
	BusInfo      [32]byte
	Version      uint32
	Capabilities uint32
	DeviceCaps   uint32
	Reserved     [3]uint32
}

type v4l2FmtDesc struct {
	Index       uint32
	Type        uint32
	Flags       uint32
	Description [32]byte
	Pixelformat uint32
	Reserved    [4]uint32
}

type v4l2PixFormat struct {
	Width, Height, Pixelformat, Field       uint32
	Bytesperline, Sizeimage                 uint32
	Colorspace, Priv                        uint32
	Flags, YcbcrEnc, Quantization, XferFunc uint32
}

type v4l2Format struct {
	Type uint32
	Pix  v4l2PixFormat
	Pad  [156]byte
}

const (
	iocNone  = 0
	iocWrite = 1
	iocRead  = 2
)

var (
	vidIOCQueryCap = ioctlR('V', 0, unsafe.Sizeof(v4l2Capability{}))
	vidIOCGFmt     = ioctlRWC('V', 4, unsafe.Sizeof(v4l2Format{}))
	vidIOCEnumFmt  = ioctlRWC('V', 2, unsafe.Sizeof(v4l2FmtDesc{}))
	vidIOCSFmt     = ioctlRWC('V', 5, unsafe.Sizeof(v4l2Format{}))
)

func ioctl(dir, typ, nr, size uintptr) uintptr {
	return (dir << 30) | (typ << 8) | nr | (size << 16)
}

func ioctlR(typ, nr byte, size uintptr) uintptr {
	return ioctl(iocRead, uintptr(typ), uintptr(nr), size)
}

func ioctlRWC(typ, nr byte, size uintptr) uintptr {
	return ioctl(iocRead|iocWrite, uintptr(typ), uintptr(nr), size)
}

func ioctlPtr(fd int, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

func fourccName(code uint32) string {
	b := []byte{
		byte(code),
		byte(code >> 8),
		byte(code >> 16),
		byte(code >> 24),
	}
	return strings.TrimRight(string(b), "\x00")
}

func v4l2loopbackLoaded() bool {
	f, err := os.Open("/proc/modules")
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "v4l2loopback ") {
			return true
		}
	}
	return false
}

func findLoopbackDevice() string {
	entries, err := os.ReadDir("/sys/class/video4linux")
	if err != nil {
		return ""
	}
	for _, e := range entries {
		namePath := "/sys/class/video4linux/" + e.Name() + "/name"
		data, err := os.ReadFile(namePath)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(data))
		if strings.Contains(strings.ToLower(name), "loopback") {
			return "/dev/" + e.Name()
		}
	}
	return ""
}

func deviceName(path string) string {
	base := strings.TrimPrefix(path, "/dev/")
	data, err := os.ReadFile("/sys/class/video4linux/" + base + "/name")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func enumOutputFormats(fd int) ([]uint32, error) {
	var formats []uint32
	for index := uint32(0); ; index++ {
		desc := v4l2FmtDesc{Index: index, Type: v4l2BufTypeVideoOutput}
		if err := ioctlPtr(fd, vidIOCEnumFmt, unsafe.Pointer(&desc)); err != nil {
			if errors.Is(err, unix.EINVAL) {
				break
			}
			return nil, fmt.Errorf("VIDIOC_ENUM_FMT index %d: %w", index, err)
		}
		formats = append(formats, desc.Pixelformat)
	}
	return formats, nil
}

func pickFormat(formats []uint32) (uint32, error) {
	preferred := []uint32{v4l2PixFmtYuyv, v4l2PixFmtRgb24}
	for _, want := range preferred {
		for _, have := range formats {
			if have == want {
				return want, nil
			}
		}
	}
	if len(formats) == 0 {
		return 0, errors.New("no writable output pixel formats reported")
	}
	return formats[0], nil
}

func setFormat(fd int, pixelformat uint32) (v4l2PixFormat, error) {
	fmtStruct := v4l2Format{Type: v4l2BufTypeVideoOutput}
	if err := ioctlPtr(fd, vidIOCGFmt, unsafe.Pointer(&fmtStruct)); err != nil {
		return v4l2PixFormat{}, fmt.Errorf("VIDIOC_G_FMT: %w", err)
	}
	if pixelformat != 0 {
		fmtStruct.Pix.Pixelformat = pixelformat
	}
	fmtStruct.Pix.Width = frameWidth
	fmtStruct.Pix.Height = frameHeight
	fmtStruct.Pix.Field = v4l2FieldNone
	if err := ioctlPtr(fd, vidIOCSFmt, unsafe.Pointer(&fmtStruct)); err != nil {
		fmt.Fprintf(os.Stderr, "VIDIOC_S_FMT failed (%dx%d %s), using G_FMT defaults: %v\n",
			frameWidth, frameHeight, fourccName(pixelformat), err)
		fmtStruct = v4l2Format{Type: v4l2BufTypeVideoOutput}
		if err2 := ioctlPtr(fd, vidIOCGFmt, unsafe.Pointer(&fmtStruct)); err2 != nil {
			return v4l2PixFormat{}, fmt.Errorf("VIDIOC_G_FMT: %w", err2)
		}
	}
	return fmtStruct.Pix, nil
}

var barColors = [8][3]uint8{
	{235, 235, 235},
	{235, 235, 16},
	{16, 235, 235},
	{16, 235, 16},
	{235, 16, 235},
	{235, 16, 16},
	{16, 16, 235},
	{16, 16, 16},
}

func rgbToYuv(r, g, b uint8) (y, u, v uint8) {
	yi := int(0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b))
	ui := int(-0.169*float64(r) - 0.331*float64(g) + 0.5*float64(b) + 128)
	vi := int(0.5*float64(r) - 0.419*float64(g) - 0.081*float64(b) + 128)
	return clamp8(yi), clamp8(ui), clamp8(vi)
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func barRGB(x, width int) (r, g, b uint8) {
	idx := (x * 8) / width
	if idx > 7 {
		idx = 7
	}
	c := barColors[idx]
	return c[0], c[1], c[2]
}

func makeRGB24Frame(width, height int) []byte {
	frame := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b := barRGB(x, width)
			off := (y*width + x) * 3
			frame[off] = r
			frame[off+1] = g
			frame[off+2] = b
		}
	}
	return frame
}

func makeYUYVFrame(width, height int) []byte {
	frame := make([]byte, width*height*2)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x += 2 {
			r0, g0, b0 := barRGB(x, width)
			x1 := x + 1
			if x1 >= width {
				x1 = width - 1
			}
			r1, g1, b1 := barRGB(x1, width)
			y0, u, v := rgbToYuv(r0, g0, b0)
			y1, _, _ := rgbToYuv(r1, g1, b1)
			off := (y*width + x) * 2
			frame[off] = y0
			frame[off+1] = u
			frame[off+2] = y1
			frame[off+3] = v
		}
	}
	return frame
}

func makeFrame(pix v4l2PixFormat) []byte {
	width := int(pix.Width)
	height := int(pix.Height)
	switch pix.Pixelformat {
	case v4l2PixFmtRgb24:
		return makeRGB24Frame(width, height)
	case v4l2PixFmtYuyv:
		return makeYUYVFrame(width, height)
	default:
		return make([]byte, int(pix.Sizeimage))
	}
}

func printSkipSetup() {
	fmt.Fprintln(os.Stderr, "SKIP: v4l2loopback is not loaded.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Setup:")
	fmt.Fprintln(os.Stderr, "  sudo modprobe v4l2loopback video_nr=2 card_label=\"spidercam-loopback\" exclusive_caps=1")
	fmt.Fprintln(os.Stderr, "  lsmod | grep v4l2loopback")
	fmt.Fprintln(os.Stderr, "  ls -l /dev/video2")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Verify:")
	fmt.Fprintln(os.Stderr, "  go run . --device /dev/video2")
	fmt.Fprintln(os.Stderr, "  ffplay -f v4l2 -i /dev/video2")
}

func main() {
	device := flag.String("device", "", "v4l2loopback output device (default: auto-detect)")
	flag.Parse()

	if !v4l2loopbackLoaded() {
		printSkipSetup()
		os.Exit(2)
	}

	devPath := *device
	if devPath == "" {
		devPath = findLoopbackDevice()
	}
	if devPath == "" {
		devPath = "/dev/video2"
	}

	fd, err := unix.Open(devPath, unix.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", devPath, err)
		printSkipSetup()
		os.Exit(2)
	}
	defer unix.Close(fd)

	var cap v4l2Capability
	if err := ioctlPtr(fd, vidIOCQueryCap, unsafe.Pointer(&cap)); err != nil {
		fmt.Fprintf(os.Stderr, "VIDIOC_QUERYCAP on %s: %v\n", devPath, err)
		os.Exit(1)
	}

	caps := cap.Capabilities
	if cap.DeviceCaps != 0 {
		caps = cap.DeviceCaps
	}
	if caps&v4l2CapVideoOutput == 0 {
		name := deviceName(devPath)
		fmt.Fprintf(os.Stderr, "SKIP: %s (%q) is not a VIDEO_OUTPUT device\n", devPath, name)
		printSkipSetup()
		os.Exit(2)
	}

	formats, err := enumOutputFormats(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enum formats: %v\n", err)
		os.Exit(1)
	}
	for _, f := range formats {
		_ = f
	}

	pixelformat, err := pickFormat(formats)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	pix, err := setFormat(fd, pixelformat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	frame := makeFrame(pix)
	if len(frame) == 0 {
		fmt.Fprintf(os.Stderr, "unsupported pixel format %s\n", fourccName(pix.Pixelformat))
		os.Exit(1)
	}
	if int(pix.Sizeimage) > 0 && len(frame) != int(pix.Sizeimage) {
		if len(frame) < int(pix.Sizeimage) {
			padded := make([]byte, pix.Sizeimage)
			copy(padded, frame)
			frame = padded
		} else {
			frame = frame[:pix.Sizeimage]
		}
	}

	fmt.Printf("device=%s name=%q format=%s %dx%d sizeimage=%d\n",
		devPath, deviceName(devPath), fourccName(pix.Pixelformat), pix.Width, pix.Height, pix.Sizeimage)

	interval := time.Second / frameFPS
	start := time.Now()

	for i := 0; i < frameCount; i++ {
		target := start.Add(time.Duration(i) * interval)
		if sleep := time.Until(target); sleep > 0 {
			time.Sleep(sleep)
		}

		n, err := unix.Write(fd, frame)
		if err != nil {
			fmt.Fprintf(os.Stderr, "write frame %d: %v\n", i, err)
			os.Exit(1)
		}
		if n != len(frame) {
			fmt.Fprintf(os.Stderr, "short write frame %d: %d/%d bytes\n", i, n, len(frame))
			os.Exit(1)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("wrote %d frames in %s (target %dfps)\n", frameCount, elapsed.Round(time.Millisecond), frameFPS)
}
