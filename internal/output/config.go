package output

import (
	"fmt"
	"os"
)

const (
	DefaultAudioSink = "spidercam_sink"
	DefaultWidth     = 1280
	DefaultHeight    = 720

	LoopbackModprobeModule = "v4l2loopback"
	LoopbackSetup          = `sudo modprobe v4l2loopback video_nr=2 card_label="spidercam-loopback" exclusive_caps=1`
	SetupSubcommand        = "spidercamd setup"
)

var LoopbackModprobeOptions = []string{
	"video_nr=2",
	"card_label=spidercam-loopback",
	"exclusive_caps=1",
}

func NullSinkModuleArgs(sinkName string) string {
	return fmt.Sprintf(
		"sink_name=%s sink_properties=device.description=Spidercam_Virtual_Mic",
		sinkName,
	)
}

type Config struct {
	Mock        bool
	VideoDevice string
	AudioSink   string
	Width       int
	Height      int
}

func DefaultConfig() Config {
	return Config{
		AudioSink: envOr("SPIDERCAM_AUDIO_SINK", DefaultAudioSink),
		VideoDevice: os.Getenv("SPIDERCAM_VIDEO_DEVICE"),
		Width:       DefaultWidth,
		Height:      DefaultHeight,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
