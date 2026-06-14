# P5.1 — PipeWire device enumeration

## Purpose

Spike **libpipewire-0.3** registry enumeration: list PipeWire **Audio/Source** and **Audio/Sink** nodes and print one JSON object per line for downstream capture routing (Wave 5 native I/O).

Output shape (one object per line):

```json
{"kind":"source","id":"42","label":"…"}
{"kind":"sink","id":"43","label":"…"}
```

## Dependencies (apt)

Runtime (PipeWire session — already present on this machine):

- `pipewire`
- `pipewire-bin` (provides `pw-cli`, optional sanity check)

Build:

- `libpipewire-0.3-dev` (pulls `libspa-0.2-dev`, `libpipewire-0.3-0t64`)

Install:

```bash
sudo apt-get install -y libpipewire-0.3-dev
```

## Build

```bash
cd experiments/wave5/p5.1-pw-list
make
```

Without system dev headers (reuse P5.2 extracted tree):

```bash
make PW_PREFIX="../p5.2-pw-capture/.pw-dev/usr"
```

## Run

```bash
./pw_list
```

## RESULT

**Status: pass** (2026-06-14, built with `PW_PREFIX` from P5.2)

```
{"kind":"sink","id":"57","label":"Ryzen HD Audio Controller Speaker"}
{"kind":"source","id":"58","label":"Ryzen HD Audio Controller Stereo Microphone"}
{"kind":"source","id":"59","label":"Ryzen HD Audio Controller Digital Microphone"}
{"kind":"sink","id":"82","label":"Spidercam_Virtual_Mic"}
```

At least one source and one sink — **pass**.

Earlier **skip** on this host: system `libpipewire-0.3-dev` not installed; fixed `pw_registry_destroy` → `spa_hook_remove` for current PipeWire API.
