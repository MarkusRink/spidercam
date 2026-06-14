# Wave 5 — Native I/O experiments

## Summary (2026-06-14)

| ID   | Verdict  | Notes                                                |
| ---- | -------- | ---------------------------------------------------- |
| P5.1 | **pass** | PipeWire JSON enum                                   |
| P5.2 | **pass** | Mic ring + pthread loop                              |
| P5.3 | **pass** | Monitor stream                                       |
| P5.4 | **pass** | `blackjack/webcam`                                   |
| P5.5 | **pass** | Loopback on `/dev/video4`; see ioctl fixes in README |
| P5.6 | **pass** | `github.com/jfreymuth/pulse`                         |
| P5.7 | **pass** | libx264 baseline IDR                                 |
| P5.8 | **pass** | Preview binary framing in `internal/preview`         |
| P5.9 | **pass** | Sink reopen < 500 ms                                 |

**All Wave 5 proofs green.**

## Operator setup (this machine)

```bash
sudo apt install -y libpipewire-0.3-dev libx264-dev
sudo modprobe v4l2loopback video_nr=2 card_label="spidercam-loopback" exclusive_caps=1
pactl load-module module-null-sink sink_name=spidercam_sink sink_properties=device.description=Spidercam_Virtual_Mic
```

Loopback may not occupy `/dev/video2` if already in use — use `go run .` in p5.5 (auto-detect) or check sysfs names.

## Findings for production

1. **PipeWire** — iterate loop during `pw_core_sync`; `spa_hook_remove` on teardown.
2. **Pulse** — `github.com/jfreymuth/pulse` (not `pulseaudio`).
3. **v4l2 caps** — `V4L2_CAP_VIDEO_OUTPUT` is `0x2`; `VIDIOC_QUERYCAP` is read-only ioctl.
4. **Loopback device path** — discover via sysfs, not hardcoded `/dev/video2`.
5. **S_FMT** — may need G_FMT fallback or struct size audit for production `internal/output/v4l2.go`.
