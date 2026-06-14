//go:build cgo && linux && spidercam_native_capture

package capture

/*
#cgo pkg-config: libpipewire-0.3
#cgo CFLAGS: -I${SRCDIR}/native
#include <stdlib.h>
#include "native/sp_capture.h"
#include "native/sp_ring.c"
#include "native/sp_capture.c"
#include "native/sp_devices.h"
#include "native/sp_devices.c"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type nativeHandle struct {
	cap *C.sp_capture
}

func (h nativeHandle) active() bool {
	return h.cap != nil
}

func listPipeWireDevices() (mics, sinks []DeviceInfo, err error) {
	mics, err = listPWKind("source")
	if err != nil {
		return nil, nil, err
	}
	sinks, err = listPWKind("sink")
	if err != nil {
		return nil, nil, err
	}
	return mics, sinks, nil
}

func listPWKind(kind string) ([]DeviceInfo, error) {
	buf := make([]C.sp_device, maxPipeWireDevices)
	var n C.int
	if kind == "source" {
		n = C.sp_list_sources(&buf[0], C.int(len(buf)))
	} else {
		n = C.sp_list_sinks(&buf[0], C.int(len(buf)))
	}
	if n < 0 {
		return nil, fmt.Errorf("pipewire list %s failed", kind)
	}
	out := make([]DeviceInfo, 0, int(n))
	for i := 0; i < int(n); i++ {
		out = append(out, DeviceInfo{
			ID:    C.GoString(&buf[i].id[0]),
			Label: C.GoString(&buf[i].label[0]),
		})
	}
	return out, nil
}

func openNativeCapture(micID, sinkID string, sampleRate int) (nativeHandle, error) {
	micC := cStringOrNil(micID)
	sinkC := cStringOrNil(sinkID)
	defer freeCString(micC)
	defer freeCString(sinkC)

	cap := C.sp_capture_open(micC, sinkC, C.int(sampleRate))
	if cap == nil {
		return nativeHandle{}, fmt.Errorf("pipewire capture open failed")
	}
	return nativeHandle{cap: cap}, nil
}

func closeNativeCapture(h nativeHandle) {
	if h.cap != nil {
		C.sp_capture_close(h.cap)
	}
}

func readNativeMic(h nativeHandle, buf []float32) int {
	if h.cap == nil || len(buf) == 0 {
		return 0
	}
	return int(C.sp_capture_read_mic(h.cap, (*C.float)(unsafe.Pointer(&buf[0])), C.int(len(buf))))
}

func readNativeMonitor(h nativeHandle, buf []float32) int {
	if h.cap == nil || len(buf) == 0 {
		return 0
	}
	return int(C.sp_capture_read_monitor(h.cap, (*C.float)(unsafe.Pointer(&buf[0])), C.int(len(buf))))
}

func cStringOrNil(s string) *C.char {
	if s == "" {
		return nil
	}
	return C.CString(s)
}

func freeCString(s *C.char) {
	if s != nil {
		C.free(unsafe.Pointer(s))
	}
}
