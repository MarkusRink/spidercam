---
name: paddleocr
description: "Local CPU OCR via PaddleOCR mobile (labels, DE)."
version: 1.0.0
author: Markus Rink
license: Apache-2.0
platforms: [linux]
metadata:
  hermes:
    tags: [OCR, PaddleOCR, CPU, Documents, Labels, German, Serial-Numbers]
    category: productivity
    related_skills: [ocr-and-documents]
    requires_toolsets: [terminal]
---

# PaddleOCR (local CPU on Hermes host)

On-device OCR with **PP-OCRv5 mobile** models. Runs on the same machine as Hermes (MINIX Z100 / N100, CPU-only). No GPU, no cloud API token, no Spiel-PC hop.

Prefer this skill for photos of labels, serial plates, calendar shots, and simple document images. For text PDFs use `ocr-and-documents` (pymupdf). For heavy layout VLM work, do not use PaddleOCR-VL on the N100.

`${HERMES_SKILL_DIR}` expands to this skill's directory.

## Install into Hermes

From this repo (or a checkout on the MINIX):

```bash
mkdir -p ~/.hermes/skills/productivity
cp -a hermes-skills/productivity/paddleocr ~/.hermes/skills/productivity/
# or symlink:
# ln -s "$(pwd)/hermes-skills/productivity/paddleocr" ~/.hermes/skills/productivity/paddleocr
```

Confirm with `hermes skills list` / `skills_list()` — look for `paddleocr`.

## When to Use

| Task | Pipeline |
|------|----------|
| Serial number / type plate / label | General OCR (`run_ocr.py` or `paddleocr ocr`) |
| Calendar dates from a photo | General OCR, then LLM post-process |
| Layout-aware document sort | PP-StructureV3 via `paddlex` (needs `paddleocr[doc-parser]`) |

**Do not use** when: remote URL extraction works via `web_extract`, or the file is a text PDF (use pymupdf).

## Prerequisites (one-time on the MINIX)

```bash
python -m pip install paddlepaddle==3.2.0 -i https://www.paddlepaddle.org.cn/packages/stable/cpu/
python -m pip install "paddleocr>=3.0.0"
# Optional layout / structure:
python -m pip install "paddleocr[doc-parser]"
```

Verify:

```bash
python -c "from paddleocr import PaddleOCR; print('ok')"
which paddleocr
```

First inference downloads models into `~/.paddlex` (a few hundred MB). Keep that cache; do not install PaddleOCR-VL weights on the N100.

If downloads fail on a restricted network, set `PADDLE_PDX_MODEL_SOURCE=BOS` or pre-seed `~/.paddlex`.

## Procedure

### 1. Prefer the helper script (stable JSON)

```bash
python ${HERMES_SKILL_DIR}/scripts/run_ocr.py /path/to/image.jpg --lang de
python ${HERMES_SKILL_DIR}/scripts/run_ocr.py /path/to/image.jpg --lang de --mobile
```

Stdout is JSON:

```json
{
  "image": "/path/to/image.jpg",
  "lang": "de",
  "results": [
    {"text": "SN-12345", "confidence": 0.98, "bbox": [[x,y],[x,y],[x,y],[x,y]]}
  ]
}
```

- Empty `results` → no text found (success, exit 0).
- `{"error": "..."}` on stderr/stdout with non-zero exit → real failure.

### 2. Direct CLI (when the script is unavailable)

```bash
paddleocr ocr -i /path/to/image.jpg --lang de --device cpu --save_path ./output
```

### 3. Layout / structure (document sorting)

```bash
paddlex --pipeline PP-StructureV3 --input /path/to/doc.jpg --device cpu --save_path ./output
```

Use layout labels (Table, Text, Title, Figure, …) and reading order to classify before deeper content work.

## Model guidance (N100)

- Default: mobile detection + lang-selected recognition (`--lang de` → Latin/German).
- `--mobile` forces `PP-OCRv5_mobile_det` (and latin mobile rec when lang is `de`/`en`/other Latin).
- Avoid server/medium models and PaddleOCR-VL on this host; RAM/CPU budget is shared with Hermes.

## Interpreting results

1. Read `results[].text` in order; join with newlines for a plain transcript.
2. For serial numbers, prefer high-confidence short alphanumeric lines; ignore low-confidence noise (`confidence < 0.5` unless nothing else matches).
3. Use `bbox` only when spatial layout matters (e.g. label fields).
4. Never invent text that is not in `results`.

## Pitfalls

- PaddleOCR **3.x** uses `predict()` / CLI `paddleocr ocr`; 2.x used `ocr()`. Stick to 3.x docs matching the installed version.
- First call is slow (model download + cache). Later calls should be ~1–2s for label-sized images on the N100.
- German is `--lang de` (also `german`). That selects the Latin-script recognition family.
- `{"error": "paddleocr is not installed..."}` → run the Prerequisites pip commands, then retry.
- Do not fall back to inventing OCR from vision when the CLI fails; report the JSON error to the user.

## Verification

```bash
python ${HERMES_SKILL_DIR}/scripts/run_ocr.py /path/to/known-label.jpg --lang de
# expect exit 0 and non-empty results[].text matching the plate
```
