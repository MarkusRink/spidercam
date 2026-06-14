# Spidercam experiments

Proof-of-concept spikes for risky integrations before production code lands in `internal/`.

Each experiment lives under `experiments/waveN/` with its own `README.md` (purpose, deps, commands, **RESULT**).

## Wave 5 — Native I/O (2026-06-14)

| ID   | Experiment                  | Directory                                           | Status   | Result                                |
| ---- | --------------------------- | --------------------------------------------------- | -------- | ------------------------------------- |
| P5.1 | PipeWire device enumeration | [wave5/p5.1-pw-list](wave5/p5.1-pw-list/)           | **pass** | JSON sources + sinks                  |
| P5.2 | PipeWire mic read           | [wave5/p5.2-pw-capture](wave5/p5.2-pw-capture/)     | **pass** | 100 frames, mic ring                  |
| P5.3 | PipeWire sink monitor       | [wave5/p5.2-pw-capture](wave5/p5.2-pw-capture/)     | **pass** | 480 samples/frame                     |
| P5.4 | v4l2 camera → PNG           | [wave5/p5.4-v4l2-cam](wave5/p5.4-v4l2-cam/)         | **pass** | `/dev/video0`                         |
| P5.5 | v4l2loopback color bars     | [wave5/p5.5-loopback](wave5/p5.5-loopback/)         | **pass** | `/dev/video4` auto-detect, 300 frames |
| P5.6 | PulseAudio null-sink tone   | [wave5/p5.6-pulse](wave5/p5.6-pulse/)               | **pass** | `jfreymuth/pulse`                     |
| P5.7 | libx264 solid frame         | [wave5/p5.7-x264](wave5/p5.7-x264/)                 | **pass** | Baseline 1280×720 IDR                 |
| P5.8 | Preview WS framing          | [wave5/p5.8-preview-pack](wave5/p5.8-preview-pack/) | **pass** | `internal/preview/pack_test.go`       |
| P5.9 | capture Reopen              | [wave5/p5.9-reopen](wave5/p5.9-reopen/)             | **pass** | Reopen 11.82 ms                       |

**Score: 9 pass, 0 skip, 0 fail**

Wave 5 index: [wave5/README.md](wave5/README.md).

Decisions: [plan/DECISIONS.md](../plan/DECISIONS.md) (D21–D29).

### Production artifacts

| Path                            | From |
| ------------------------------- | ---- |
| `go.mod`                        | P5.8 |
| `internal/preview/pack.go`      | P5.8 |
| `internal/preview/pack_test.go` | P5.8 |

## Result legend

- **pass** — acceptance criteria met
- **skip** — deps or hardware missing; documented workaround
- **fail** — ran but criteria not met
