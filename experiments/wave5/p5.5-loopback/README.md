# P5.5 — v4l2loopback color bars

Writes 300 SMPTE-style color-bar frames at 30 fps to a v4l2loopback `VIDEO_OUTPUT` device.

## Prerequisites

```bash
lsmod | grep v4l2loopback
```

If empty, load the module:

```bash
sudo modprobe v4l2loopback video_nr=2 card_label="spidercam-loopback" exclusive_caps=1
```

**Note:** If `/dev/video2` is already taken (e.g. integrated camera metadata node), loopback may appear as `/dev/video4` or another free number. The program **auto-detects** sysfs name containing `loopback`.

```bash
grep -l loopback /sys/class/video4linux/video*/name | xargs -I{} dirname {} | xargs -I{} basename {}
```

## Build & run

```bash
cd experiments/wave5/p5.5-loopback
go run .
```

Optional explicit device:

```bash
go run . --device /dev/video4
```

| Flag       | Default              | Description              |
| ---------- | -------------------- | ------------------------ |
| `--device` | auto-detect loopback | v4l2loopback output node |

The program:

1. Finds loopback node by sysfs name (or uses `--device`)
2. `VIDIOC_QUERYCAP` — requires `V4L2_CAP_VIDEO_OUTPUT` (`0x00000002`)
3. `VIDIOC_ENUM_FMT` / `VIDIOC_G_FMT` / `VIDIOC_S_FMT` (falls back to G_FMT defaults if 1280×720 set fails)
4. `write()` × 300 frames @ 30 fps

## Verify with ffplay

```bash
ffplay -f v4l2 -video_size 1280x720 -i /dev/video4
```

(Run while `go run .` is writing, or use a second terminal.)

## RESULT

**Status: pass** (2026-06-14)

| Field  | Value                                                                                                                |
| ------ | -------------------------------------------------------------------------------------------------------------------- |
| Device | `/dev/video4` (`spidercam-loopback`)                                                                                 |
| Module | `v4l2loopback` loaded                                                                                                |
| Frames | 300 in ~10 s @ 30 fps target                                                                                         |
| Fixes  | `V4L2_CAP_VIDEO_OUTPUT` = `0x2` (not `0x20000`); `VIDIOC_QUERYCAP` ioctl direction `_IOR`; auto-detect loopback path |

`video_nr=2` did not replace integrated camera on `/dev/video2`; loopback landed on `/dev/video4`.

Sample run:

```
device=/dev/video4 name="spidercam-loopback" format=YUYV 640x480 sizeimage=...
wrote 300 frames in 9.968s (target 30fps)
```

Typo reminder: flag is `--device`, not `--deice`.
