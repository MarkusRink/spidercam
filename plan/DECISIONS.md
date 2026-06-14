# Open decisions

Status: **🔴 open** | **🟡 proposed default** | **🟢 resolved**

---

## D1 — Host vs participant state channel

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d1)

---

## D3 — Frame tap implementation

**🟢 Resolved** — Go pull engine; no browser AudioWorklet. See [audio/frame-engine.md](./audio/frame-engine.md).

---

## D5 — Latency measurement

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d5)

---

## D6 — CI / E2E

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d6)

---

## D8 — Host VAD special handling

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d8)

---

## D11 — Capture backend (Linux)

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d11)

---

## D12 — Reference ducking default

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d12)

---

## D15 — Host UI layout & session config

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d15)

---

## D16 — Host preview stream

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d16)

---

## D17 — Dual-branch audio pipeline

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d17)

---

## D18 — Per-stream AEC

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d18)

---

## D19 — Per-stream denoise

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d19)

---

## Resolved elsewhere

D2, D4, D5, D6, D7, D8, D9, D10, D11, D12, D13, D15, D16, D17, D18, D19, D20, D21–D30 (Go architecture + Wave 5 native I/O) → [DECISIONS.resolved.md](./DECISIONS.resolved.md)

Wave 5 proof evidence: [experiments/wave5/README.md](../experiments/wave5/README.md) (P5.1–P5.9, all pass 2026-06-14).

---

## Wave 5 — Native I/O

Validated by [experiments/wave5/](../experiments/wave5/) before `internal/capture`, `internal/output`, `internal/preview` production work.

### D21 — WebRTC SDP negotiation direction

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d21--webrtc-sdp-negotiation-direction)

---

### D22 — PulseAudio virtual mic integration

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d22--pulseaudio-virtual-mic)

---

### D23 — v4l2 camera capture library

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d23--v4l2-camera-library)

---

### D24 — v4l2loopback output pixel format

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d24--v4l2loopback-pixel-format)

---

### D25 — PipeWire thread model

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d25--pipewire-thread-model)

---

### D26 — libx264 in CI

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d26--libx264-in-ci)

---

### D27 — `!cgo` build fallback

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d27--cgo-build-fallback)

---

### D28 — Virtual device provisioning

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d28--virtual-device-provisioning)

---

### D29 — Go module path

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d29--go-module-path)

---

### D30 — Virtual cam device path

**🟢 Resolved** → [DECISIONS.resolved.md](./DECISIONS.resolved.md#d30--virtual-cam-device-path)

---

## How to close a decision

1. Discuss referencing decision ID
2. Move to `DECISIONS.resolved.md`
3. Update affected sub-specs
