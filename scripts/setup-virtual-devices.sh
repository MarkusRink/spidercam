#!/usr/bin/env bash
set -euo pipefail

echo "spidercam virtual device setup (Linux)"
echo ""

if ! command -v modprobe &>/dev/null; then
  echo "modprobe not found — run as root on a system with kernel modules"
  exit 1
fi

echo "→ loading v4l2loopback (virtual webcam at /dev/video2)"
if ! lsmod | grep -q v4l2loopback; then
  sudo modprobe v4l2loopback devices=1 video_nr=2 card_label="spidercam" exclusive_caps=1
fi

if command -v pactl &>/dev/null; then
  echo "→ creating PulseAudio null sink spidercam_sink"
  if ! pactl list short sinks | grep -q spidercam_sink; then
    pactl load-module module-null-sink sink_name=spidercam_sink sink_properties=device.description=spidercam_sink
  fi
  if ! pactl list short sources | grep -q spidercam_source; then
    pactl load-module module-remap-source master=spidercam_sink.monitor source_name=spidercam_source source_properties=device.description=spidercam_mic
  fi
  echo ""
  echo "Teams setup:"
  echo "  camera: /dev/video2 (spidercam)"
  echo "  microphone: spidercam_mic (PulseAudio source)"
elif command -v pw-cli &>/dev/null; then
  echo "→ PipeWire detected — create a virtual sink/source in pw-top or use:"
  echo "  pw-cli create-node adapter ... (see README)"
else
  echo "⚠ no PulseAudio or PipeWire found"
fi

echo ""
echo "done"
