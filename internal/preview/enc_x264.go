//go:build cgo && linux

package preview

/*
#cgo pkg-config: x264
#include "native/enc_x264.h"
#include "native/enc_x264.c"
*/
import "C"
import (
	"errors"
	"time"
	"unsafe"
)

type x264Encoder struct {
	enc *C.sp_x264_enc
}

func openX264(cfg Config) (Encoder, error) {
	enc := C.sp_x264_open(C.int(cfg.Width), C.int(cfg.Height), C.int(cfg.FPS), C.int(cfg.BitrateKbps))
	if enc == nil {
		return nil, errors.New("preview: x264_encoder_open failed")
	}
	return &x264Encoder{enc: enc}, nil
}

func (e *x264Encoder) Encode(rgba []byte, w, h int, ts time.Time) (EncodeResult, error) {
	if len(rgba) == 0 {
		return EncodeResult{}, errors.New("preview: empty rgba frame")
	}
	var out *C.uint8_t
	var outLen C.int
	var isKey C.int
	rc := C.sp_x264_encode(
		e.enc,
		(*C.uint8_t)(unsafe.Pointer(&rgba[0])),
		C.int(w),
		C.int(h),
		C.int64_t(ts.UnixMicro()),
		&out,
		&outLen,
		&isKey,
	)
	if rc != 0 {
		return EncodeResult{}, errors.New("preview: x264 encode failed")
	}
	if outLen <= 0 {
		return EncodeResult{}, nil
	}
	avcc := C.GoBytes(unsafe.Pointer(out), outLen)
	return EncodeResult{AVCC: avcc, Keyframe: isKey != 0}, nil
}

func (e *x264Encoder) ForceKeyframe() {
	C.sp_x264_force_keyframe(e.enc)
}

func (e *x264Encoder) Close() error {
	C.sp_x264_close(e.enc)
	e.enc = nil
	return nil
}
