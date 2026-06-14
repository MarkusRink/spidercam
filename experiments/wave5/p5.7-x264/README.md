# P5.7 — libx264 solid red 1280×720 IDR

Mini proof: fill RGBA red, convert to I420, encode one IDR access unit with libx264 (`ultrafast` + `zerolatency`, H.264 **baseline**, Annex-B) and dump AVCC-style length-prefixed NALs for P5.8.

Encoder settings align with [plan/architecture/preview.md](../../../plan/architecture/preview.md).

## Prerequisites

```bash
sudo apt install libx264-dev pkg-config build-essential
```

Optional verify playback:

```bash
sudo apt install ffmpeg   # provides ffprobe, ffplay
```

## Build and run

```bash
cd experiments/wave5/p5.7-x264
make
./enc_test
```

Outputs:

| File                 | Format                                                                         |
| -------------------- | ------------------------------------------------------------------------------ |
| `output/test.h264`   | Annex-B byte stream                                                            |
| `output/sample.avcc` | First access unit: big-endian `uint32` NAL length + NAL bytes (no start codes) |

## Verify

```bash
ffprobe -hide_banner output/test.h264
ffplay -autoexit output/test.h264   # solid red frame
```

Acceptance: `ffprobe` reports `h264`, and the stream contains an **IDR** slice (keyframe). Hex peek at AVCC sample:

```bash
xxd -l 64 output/sample.avcc
```

## Acceptance

- `make && ./enc_test` succeeds.
- `ffprobe` shows H.264 with IDR/keyframe.
- If `libx264-dev` is missing, **skip** (Makefile errors with install hint).

## RESULT

**Status: pass** (2026-06-14)

| Field   | Value                                            |
| ------- | ------------------------------------------------ |
| Library | libx264 0.165 (`libx264-dev`)                    |
| Output  | `output/test.h264`, `output/sample.avcc`         |
| Profile | Constrained Baseline 1280×720 IDR                |
| ffprobe | `h264 (Constrained Baseline), yuv420p, 1280x720` |

```
wrote output/test.h264 and output/sample.avcc (1280x720 IDR)
x264 [info]: profile Constrained Baseline, level 3.1, 4:2:0, 8-bit
```
