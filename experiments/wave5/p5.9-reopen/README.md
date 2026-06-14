# P5.9 — capture Reopen (sink switch)

Close and reopen `sp_capture` on a different sink without segfault. Measures reopen latency.

## Build

Requires PipeWire runtime (user session) and dev headers:

```bash
sudo apt install libpipewire-0.3-dev
make
```

If `pkg-config` cannot find headers (runtime only), reuse P5.2 extracted dev tree:

```bash
make PW_PREFIX="../p5.2-pw-capture/.pw-dev/usr"
```

## Run

```bash
./reopen_test
```

Flow:

1. Enumerate sinks via PipeWire registry (P5.1 pattern)
2. Open default mic + sink A, read 50 monitor RMS frames
3. Close, reopen default mic + sink B (or same sink if only one)
4. Read 50 more frames, report `reopen_ms`

## Acceptance

| Criterion        | Threshold   |
| ---------------- | ----------- |
| Reopen completes | < 500 ms    |
| Process exit     | no segfault |

If only one sink exists: skip sink-switch verdict, still exercise close/reopen on that sink.

## Files

| File             | Role                                   |
| ---------------- | -------------------------------------- |
| `sp_ring.h/c`    | SPSC ring (from P5.2)                  |
| `sp_capture.h/c` | PW pthread capture (from P5.2)         |
| `reopen_test.c`  | enumerate → capture → reopen → capture |

## RESULT

**Status: pass** (2026-06-14)

| Criterion                   | Outcome                                                     |
| --------------------------- | ----------------------------------------------------------- |
| Two sinks enumerated        | pass — id 57 (Ryzen Speaker), id 82 (Spidercam_Virtual_Mic) |
| 50 frames phase 1 + phase 2 | pass — 480 samples/frame after warmup                       |
| Reopen < 500 ms             | pass — `reopen_ms=11.82`                                    |
| No segfault                 | pass — exit 0                                               |

Sample log:

```
P5.9 reopen test — enumerate sinks
  sink[0] id=57 label=Ryzen HD Audio Controller Speaker
  sink[1] id=82 label=Spidercam_Virtual_Mic
sink A=57  sink B=82
phase 1: open mic=(default) sink=57
  phase1 frame  0  mon_rms=0.000000 (0)
  phase1 frame 48  mon_rms=0.000000 (480)
  phase1 avg mon_rms=0.000000 over 50 frames
closing capture...
phase 2: reopen mic=(default) sink=82
  phase2 frame  0  mon_rms=0.000000 (0)
  phase2 frame 48  mon_rms=0.000000 (480)
  phase2 avg mon_rms=0.000000 over 50 frames
reopen_ms=11.82
PASS: sink switch reopen_ms=11.82, no crash
```

Built with `PW_PREFIX` from P5.2 (no system `libpipewire-0.3-dev`). `mon_rms` is 0 without playback to the sink — expected per P5.2.

Not integrated into `spidercamd/internal/` per scope.
