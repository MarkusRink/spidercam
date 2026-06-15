//go:build linux

package output

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	v4l2BufTypeVideoOutput = 2
	v4l2FieldNone          = 1
	v4l2CapVideoOutput     = 0x00000002

	v4l2PixFmtYuyv  = 0x56595559
	v4l2PixFmtRgb24 = 0x33424752
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

type v4l2Writer struct {
	fd      int
	pix     v4l2PixFormat
	healthy bool
}

func openV4L2Output(path string, width, height int) (*v4l2Writer, error) {
	fd, err := unix.Open(path, unix.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	var cap v4l2Capability
	if err := ioctlPtr(fd, vidIOCQueryCap, unsafe.Pointer(&cap)); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("VIDIOC_QUERYCAP on %s: %w", path, err)
	}

	caps := cap.Capabilities
	if cap.DeviceCaps != 0 {
		caps = cap.DeviceCaps
	}
	if caps&v4l2CapVideoOutput == 0 {
		_ = unix.Close(fd)
		name := deviceCardName(path)
		return nil, fmt.Errorf("%s (%q) is not a VIDEO_OUTPUT device", path, name)
	}

	formats, err := enumOutputFormats(fd)
	if err != nil {
		_ = unix.Close(fd)
		return nil, err
	}

	pixelformat, err := pickFormat(formats)
	if err != nil {
		_ = unix.Close(fd)
		return nil, err
	}

	pix, err := setFormat(fd, pixelformat, width, height)
	if err != nil {
		_ = unix.Close(fd)
		return nil, err
	}

	return &v4l2Writer{fd: fd, pix: pix, healthy: true}, nil
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

func setFormat(fd int, pixelformat uint32, width, height int) (v4l2PixFormat, error) {
	trySizes := [][2]int{{width, height}, {640, 480}, {1280, 720}}
	seen := make(map[[2]int]bool)
	var sizes [][2]int
	for _, s := range trySizes {
		if s[0] <= 0 || s[1] <= 0 || seen[s] {
			continue
		}
		seen[s] = true
		sizes = append(sizes, s)
	}

	var lastErr error
	for _, size := range sizes {
		pix, err := trySetFormat(fd, pixelformat, size[0], size[1])
		if err != nil {
			lastErr = err
			continue
		}
		if pix.Width == 0 || pix.Height == 0 || pix.Sizeimage == 0 {
			lastErr = fmt.Errorf("invalid format %dx%d sizeimage=%d", pix.Width, pix.Height, pix.Sizeimage)
			continue
		}
		return pix, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no usable output format")
	}
	return v4l2PixFormat{}, lastErr
}

func trySetFormat(fd int, pixelformat uint32, width, height int) (v4l2PixFormat, error) {
	fmtStruct := v4l2Format{Type: v4l2BufTypeVideoOutput}
	if err := ioctlPtr(fd, vidIOCGFmt, unsafe.Pointer(&fmtStruct)); err != nil {
		return v4l2PixFormat{}, fmt.Errorf("VIDIOC_G_FMT: %w", err)
	}
	if pixelformat != 0 {
		fmtStruct.Pix.Pixelformat = pixelformat
	}
	fmtStruct.Pix.Width = uint32(width)
	fmtStruct.Pix.Height = uint32(height)
	fmtStruct.Pix.Field = v4l2FieldNone
	if err := ioctlPtr(fd, vidIOCSFmt, unsafe.Pointer(&fmtStruct)); err != nil {
		return v4l2PixFormat{}, fmt.Errorf("VIDIOC_S_FMT %dx%d: %w", width, height, err)
	}
	return fmtStruct.Pix, nil
}

func (w *v4l2Writer) WriteVideo(rgba []byte, width, height int) error {
	if !w.healthy {
		return errors.New("v4l2 output unhealthy")
	}
	dw := int(w.pix.Width)
	dh := int(w.pix.Height)
	if width != dw || height != dh {
		rgba = resizeRGBA(rgba, width, height, dw, dh)
	}
	frame, err := convertRGBA(rgba, dw, dh, w.pix)
	if err != nil {
		return err
	}
	n, err := unix.Write(w.fd, frame)
	if err != nil {
		w.healthy = false
		return err
	}
	if n != len(frame) {
		w.healthy = false
		return fmt.Errorf("short write: %d/%d bytes", n, len(frame))
	}
	return nil
}

func (w *v4l2Writer) Healthy() bool {
	return w.healthy
}

func (w *v4l2Writer) Close() error {
	if w.fd >= 0 {
		err := unix.Close(w.fd)
		w.fd = -1
		return err
	}
	return nil
}

func convertRGBA(rgba []byte, width, height int, pix v4l2PixFormat) ([]byte, error) {
	switch pix.Pixelformat {
	case v4l2PixFmtRgb24:
		frame := rgbaToRGB24(rgba, width, height)
		return padFrame(frame, pix), nil
	case v4l2PixFmtYuyv:
		frame := rgbaToYUYV(rgba, width, height)
		return padFrame(frame, pix), nil
	default:
		size := int(pix.Sizeimage)
		if size <= 0 {
			size = width * height * 4
		}
		frame := make([]byte, size)
		return frame, nil
	}
}

func padFrame(frame []byte, pix v4l2PixFormat) []byte {
	if int(pix.Sizeimage) <= 0 {
		return frame
	}
	if len(frame) == int(pix.Sizeimage) {
		return frame
	}
	if len(frame) < int(pix.Sizeimage) {
		padded := make([]byte, pix.Sizeimage)
		copy(padded, frame)
		return padded
	}
	return frame[:pix.Sizeimage]
}

func resizeRGBA(src []byte, sw, sh, dw, dh int) []byte {
	if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
		return src
	}
	dst := make([]byte, dw*dh*4)
	for y := 0; y < dh; y++ {
		sy := y * sh / dh
		for x := 0; x < dw; x++ {
			sx := x * sw / dw
			srcOff := (sy*sw + sx) * 4
			dstOff := (y*dw + x) * 4
			copy(dst[dstOff:dstOff+4], src[srcOff:srcOff+4])
		}
	}
	return dst
}

func rgbaToRGB24(rgba []byte, width, height int) []byte {
	frame := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offRGBA := (y*width + x) * 4
			offRGB := (y*width + x) * 3
			frame[offRGB] = rgba[offRGBA]
			frame[offRGB+1] = rgba[offRGBA+1]
			frame[offRGB+2] = rgba[offRGBA+2]
		}
	}
	return frame
}

func rgbaToYUYV(rgba []byte, width, height int) []byte {
	frame := make([]byte, width*height*2)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x += 2 {
			r0, g0, b0 := rgbaPixel(rgba, width, x, y)
			x1 := x + 1
			if x1 >= width {
				x1 = width - 1
			}
			r1, g1, b1 := rgbaPixel(rgba, width, x1, y)
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

func rgbaPixel(rgba []byte, width, x, y int) (r, g, b uint8) {
	off := (y*width + x) * 4
	if off+2 >= len(rgba) {
		return 0, 0, 0
	}
	return rgba[off], rgba[off+1], rgba[off+2]
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
