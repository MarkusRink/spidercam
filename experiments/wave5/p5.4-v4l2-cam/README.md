# P5.4 — v4l2 camera → PNG

Mini proof: list `/dev/video*` with card names, open a capture device, read one frame as RGBA, save PNG.

## Library choice

**`github.com/blackjack/webcam`** (D23 option B) — not raw `ioctl`.

| Concern                               | Choice                                                                       |
| ------------------------------------- | ---------------------------------------------------------------------------- |
| Device listing (card names)           | sysfs `/sys/class/video4linux/<n>/name`                                      |
| Open / format negotiation / streaming | `blackjack/webcam` (`Open`, `SetImageFormat`, `StartStreaming`, `ReadFrame`) |
| YUYV → RGBA                           | `image.YCbCr` + `draw.Draw` into `image.RGBA`                                |
| MJPEG → RGBA                          | `image/jpeg.Decode` + `draw.Draw`                                            |

Raw `VIDIOC_*` ioctl would duplicate what the library already wraps; sysfs is simpler than ioctl for enumeration. Production target: `internal/capture/v4l2_camera.go`.

## Dependencies

- Linux with v4l2 (`/dev/video*`)
- Go 1.22+
- `github.com/blackjack/webcam` (pulled by `go mod tidy`)
- User in `video` group (or root) for device access

## Run

```bash
cd experiments/wave5/p5.4-v4l2-cam
go mod tidy
go run .                          # first working capture device
go run . --device /dev/video0     # explicit device
```

Output: `output/frame.png`

## Acceptance

- **pass** — PNG exists, > 5 KiB (non-trivial frame)
- **skip** — no `/dev/video*` nodes, or none openable / no MJPEG/YUYV at supported resolution
- **fail** — ran but PNG missing, too small, or capture error

## RESULT

**pass** — 2026-06-14 on dev machine

| Field  | Value                                           |
| ------ | ----------------------------------------------- |
| Device | `/dev/video0` (Integrated Camera: Integrated C) |
| Format | YUYV 4:2:2 640×480                              |
| Output | `output/frame.png` — 313 027 bytes              |
