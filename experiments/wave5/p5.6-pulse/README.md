# P5.6 — PulseAudio null-sink tone

Mini proof: 3s mono float32 440Hz sine @ 48kHz into virtual mic sink `spidercam_sink`.

Decision D22 in [docs/DECISIONS.md](../../../docs/DECISIONS.md) names `github.com/jfreymuth/pulseaudio`; that module does not exist. This proof uses **`github.com/jfreymuth/pulse`** (pure Go, no CGO), which is the maintained PulseAudio client.

## Prerequisites

- PulseAudio or PipeWire with PulseAudio compatibility (`pactl`, `parec`)
- Go 1.22+

## Create null sink (if missing)

```bash
pactl list sinks short | grep -q spidercam_sink || \
  pactl load-module module-null-sink \
    sink_name=spidercam_sink \
    sink_properties=device.description=Spidercam_Virtual_Mic
```

Unload when done:

```bash
pactl unload-module "$(pactl list modules short | awk '/sink_name=spidercam_sink/ {print $1; exit}')"
```

## Build and run

```bash
cd experiments/wave5/p5.6-pulse
go mod tidy
go run .
```

## Verify (parec)

Record from the sink monitor while the tone plays (second terminal):

```bash
parec --device=spidercam_sink.monitor \
  --format=float32le --rate=48000 --channels=1 \
  --file-format=raw --latency-msec=50 /tmp/p5.6-verify.raw
```

Expect ~576000 bytes (3s × 48000 × 4). Quick level check:

```bash
python3 - <<'PY'
import struct, math
data = open("/tmp/p5.6-verify.raw", "rb").read()
samples = struct.unpack(f"<{len(data)//4}f", data)
rms = math.sqrt(sum(s*s for s in samples) / len(samples))
print(f"samples={len(samples)} rms={rms:.4f} (expect ~0.707 for full-scale sine)")
PY
```

## Acceptance

- Program connects to PulseAudio, targets `spidercam_sink`, plays 3s 440Hz mono float32 @ 48kHz without error.
- If PulseAudio or the sink cannot be set up, document skip steps above.

## RESULT

| Field   | Value                                                                                                     |
| ------- | --------------------------------------------------------------------------------------------------------- |
| Status  | **pass**                                                                                                  |
| Library | `github.com/jfreymuth/pulse` validated; plan name `github.com/jfreymuth/pulseaudio` rejected (module 404) |
| Machine | Ubuntu 26.04, PipeWire Pulse compat                                                                       |
| Notes   | Null sink created via `module-null-sink`; `go run .` completed; parec monitor ~440.4Hz, rms 0.31          |
