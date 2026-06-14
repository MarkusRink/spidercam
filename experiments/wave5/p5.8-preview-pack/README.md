# P5.8 — Preview WS binary framing

Proof that `internal/preview` `PackChunk` and `AnnexBToAVCC` match the mock-server framing in `apps/mock-server/src/preview-stream.ts` and [plan/API.md](../../plan/API.md).

## Binary format

```text
[flags: u8][pts_us: u64 BE][nal_len: u32 BE][h264_au: nal_len bytes]
```

- Flag bit `0x01` = keyframe (IDR)
- `h264_au` is AVCC-style length-prefixed NALs (converted from Annex B fixture via `AnnexBToAVCC`)

## Test command

From repo root:

```bash
go test ./internal/preview/... -run Pack -count=1 -v
```

## RESULT

**pass** — 2026-06-14

```
=== RUN   TestPackChunkKeyframe
--- PASS: TestPackChunkKeyframe (0.00s)
=== RUN   TestPackChunkNonKeyframe
--- PASS: TestPackChunkNonKeyframe (0.00s)
PASS
ok  	github.com/markus/spidercam/internal/preview	0.004s
```

Asserts keyframe flag, 13-byte header length, big-endian `pts_us` and `nal_len`, and AVCC payload from `web/test-fixtures/preview/keyframe.h264`.
