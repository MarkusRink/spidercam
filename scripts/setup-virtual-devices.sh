#!/usr/bin/env bash
set -euo pipefail

failures=0
warnings=0

print_ok() { echo "  ✓ $1"; }
print_warn() { echo "  ⚠ $1 — $2"; warnings=$((warnings + 1)); }
print_fail() { echo "  ✗ $1 — $2"; failures=$((failures + 1)); }

kernel_release="$(uname -r)"
module_path=""
if [[ -d "/lib/modules/${kernel_release}" ]]; then
  module_path="$(find "/lib/modules/${kernel_release}" -name 'v4l2loopback.ko*' 2>/dev/null | head -1)"
fi

echo "spidercam virtual device setup (Linux)"
echo ""
echo "Checking requirements..."

if ! command -v modprobe &>/dev/null; then
  print_fail "kernel modules" "modprobe not found — run on a Linux system with loadable kernel modules"
else
  print_ok "kernel module support"
fi

if ! command -v sudo &>/dev/null; then
  print_fail "sudo" "required to load v4l2loopback"
else
  print_ok "sudo"
fi

if lsmod | grep -q v4l2loopback || [[ -n "${module_path}" ]]; then
  print_ok "v4l2loopback (kernel ${kernel_release})"
elif dpkg -l v4l2loopback-dkms 2>/dev/null | grep -q '^ii'; then
  print_fail "v4l2loopback" "package installed but no module for kernel ${kernel_release}"
  echo ""
  echo "    Try rebuilding the module:"
  echo "      sudo apt install --reinstall v4l2loopback-dkms"
  echo "      sudo dkms status"
else
  print_fail "v4l2loopback" "not installed for kernel ${kernel_release}"
  echo ""
  echo "    Install it with:"
  echo "      sudo apt install v4l2loopback-dkms"
fi

if command -v pactl &>/dev/null; then
  print_ok "PulseAudio (pactl)"
elif command -v pw-cli &>/dev/null; then
  print_warn "audio" "PipeWire found but pactl missing — install pipewire-pulse for automatic virtual mic setup"
else
  print_fail "audio" "neither PulseAudio (pactl) nor PipeWire (pw-cli) found"
  echo ""
  echo "    On Ubuntu, install one of:"
  echo "      sudo apt install pulseaudio-utils"
  echo "      sudo apt install pipewire-audio"
fi

echo ""

if (( failures > 0 )); then
  echo "Fix the missing requirements above, then re-run this script."
  exit 1
fi

if (( warnings > 0 )); then
  echo "Warnings present — continuing with what is available."
  echo ""
fi

echo "Setting up virtual devices..."

if ! lsmod | grep -q v4l2loopback; then
  echo "→ loading v4l2loopback (virtual webcam at /dev/video2)"
  if ! sudo modprobe v4l2loopback devices=1 video_nr=2 card_label="spidercam" exclusive_caps=1; then
    echo "✗ failed to load v4l2loopback"
    exit 1
  fi
else
  echo "→ v4l2loopback already loaded"
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
fi

echo ""
echo "done"
