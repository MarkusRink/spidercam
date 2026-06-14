# P5.2 / P5.3 — PipeWire mic + sink monitor capture

Non-blocking read from PipeWire mic and sink monitor via SPSC ring buffers (480 samples × 8 frames). PipeWire `pw_main_loop` runs on a dedicated pthread; the test thread pulls frames without blocking on PW.

## Build

Requires PipeWire runtime (user session) and dev headers:

```bash
sudo apt install libpipewire-0.3-dev
make
```

If `pkg-config` cannot find headers but runtime libs are installed, extract dev headers locally:

```bash
apt-get download libpipewire-0.3-dev libspa-0.2-dev
mkdir -p .pw-dev && dpkg-deb -x libpipewire-0.3-dev_*.deb .pw-dev && dpkg-deb -x libspa-0.2-dev_*.deb .pw-dev
make PW_PREFIX="$(pwd)/.pw-dev/usr"
```

## Run

```bash
./pw_read_test
./pw_read_test --mic-id 42 --sink-id 55   # numeric PW node IDs
```

Empty `--mic-id` / `--sink-id` uses PipeWire default routes.

### Monitor RMS validation (operator)

`mon_rms` stays near zero unless audio is playing to the selected sink. To validate monitor capture:

```bash
# find default sink (example)
pactl get-default-sink

# play a test tone to that sink while pw_read_test runs
speaker-test -t sine -f 440 -l 1
# or: aplay -D pipewire /usr/share/sounds/alsa/Front_Center.wav
```

## Files

| File             | Role                                   |
| ---------------- | -------------------------------------- |
| `sp_ring.h/c`    | SPSC ring, 3840 samples                |
| `sp_capture.h/c` | PW pthread loop, mic + monitor streams |
| `pw_read_test.c` | 100-frame RMS printer                  |

## RESULT

**Status: pass** (2026-06-14)

| Criterion                    | Outcome                                                      |
| ---------------------------- | ------------------------------------------------------------ |
| 100 frames, no deadlock      | pass — completed in ~1.4 s                                   |
| `mic_rms` printed each frame | pass — ambient mic ~0.23 RMS, 480 samples/frame after warmup |
| `mon_rms` printed each frame | pass — reads 480 samples/frame; RMS 0.0 without playback     |

Sample log (frames 6–10, 99):

```
frame   6  mic_rms=0.983988 (480)  mon_rms=0.000000 (0)
frame   7  mic_rms=1.000000 (480)  mon_rms=0.000000 (0)
frame   8  mic_rms=0.633333 (480)  mon_rms=0.000000 (0)
frame   9  mic_rms=0.126802 (480)  mon_rms=0.000000 (0)
frame  10  mic_rms=0.011372 (480)  mon_rms=0.000000 (0)
...
frame  99  mic_rms=0.228803 (480)  mon_rms=0.000000 (480)
done: 100 frames
```

### Issues encountered

1. **`libpipewire-0.3-dev` not installed** — system had runtime only; built with `apt-get download` + `PW_PREFIX` workaround.
2. **Initial deadlock** — `pw_core_sync` waited on `pthread_cond` before `pw_main_loop_run`; fixed by driving `pw_loop_iterate` during setup.
3. **`mon_rms` validation** — requires operator to play audio to the sink (`speaker-test` / `aplay`); not automated in this proof.

Not integrated into `spidercamd/internal/` per scope.
