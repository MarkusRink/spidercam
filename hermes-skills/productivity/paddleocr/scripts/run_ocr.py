#!/usr/bin/env python3
"""
Runs PaddleOCR on a single image and prints clean JSON to stdout.

Usage:
    python run_ocr.py <image_path> [--lang de] [--device cpu] [--mobile]

Output (stdout, JSON):
    {
      "image": "<path>",
      "lang": "de",
      "results": [
        {"text": "...", "confidence": 0.98, "bbox": [[x,y], [x,y], [x,y], [x,y]]},
        ...
      ]
    }

Exits non-zero with a JSON error object on failure, so the calling agent
can distinguish "no text found" (empty results list) from an actual error.
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


LATIN_LANGS = {
    "de",
    "german",
    "en",
    "fr",
    "french",
    "es",
    "it",
    "pt",
    "nl",
    "pl",
    "cs",
    "sk",
    "hu",
    "ro",
    "sv",
    "da",
    "fi",
    "no",
    "tr",
    "af",
    "la",
    "latin",
}


def _fail(message: str, code: int = 1) -> int:
    print(json.dumps({"error": message}, ensure_ascii=False))
    return code


def _to_list(value: Any) -> list[Any]:
    if value is None:
        return []
    if hasattr(value, "tolist"):
        try:
            return value.tolist()
        except Exception:
            pass
    if isinstance(value, (list, tuple)):
        return list(value)
    return [value]


def _normalize_bbox(bbox: Any) -> list[list[float]]:
    points = _to_list(bbox)
    out: list[list[float]] = []
    for point in points:
        coords = _to_list(point)
        if len(coords) >= 2:
            out.append([float(coords[0]), float(coords[1])])
    return out


def _results_from_v3_page(page: Any) -> list[dict[str, Any]]:
    data: dict[str, Any]
    if isinstance(page, dict):
        data = page
    elif hasattr(page, "json") and isinstance(page.json, dict):
        data = page.json
    elif hasattr(page, "keys"):
        try:
            data = dict(page)
        except Exception:
            return []
    else:
        return []

    texts = _to_list(data.get("rec_texts"))
    scores = _to_list(data.get("rec_scores"))
    polys = _to_list(data.get("rec_polys") or data.get("dt_polys") or data.get("rec_boxes"))

    results: list[dict[str, Any]] = []
    for i, text in enumerate(texts):
        if text is None:
            continue
        text_str = str(text).strip()
        if not text_str:
            continue
        confidence = float(scores[i]) if i < len(scores) else 0.0
        bbox = _normalize_bbox(polys[i]) if i < len(polys) else []
        results.append(
            {
                "text": text_str,
                "confidence": round(confidence, 4),
                "bbox": bbox,
            }
        )
    return results


def _results_from_v2_raw(raw: Any) -> list[dict[str, Any]]:
    results: list[dict[str, Any]] = []
    for page in raw or []:
        for line in page or []:
            try:
                bbox, (text, confidence) = line
            except (ValueError, TypeError):
                continue
            text_str = str(text).strip()
            if not text_str:
                continue
            results.append(
                {
                    "text": text_str,
                    "confidence": round(float(confidence), 4),
                    "bbox": _normalize_bbox(bbox),
                }
            )
    return results


def _build_ocr(lang: str, device: str, mobile: bool) -> Any:
    from paddleocr import PaddleOCR

    kwargs: dict[str, Any] = {
        "lang": lang,
        "device": device,
    }
    if mobile:
        kwargs["text_detection_model_name"] = "PP-OCRv5_mobile_det"
        lang_key = lang.lower()
        if lang_key in {"en", "english"}:
            kwargs["text_recognition_model_name"] = "en_PP-OCRv5_mobile_rec"
        elif lang_key in LATIN_LANGS:
            kwargs["text_recognition_model_name"] = "latin_PP-OCRv5_mobile_rec"
        else:
            kwargs["text_recognition_model_name"] = "PP-OCRv5_mobile_rec"
        kwargs["use_doc_orientation_classify"] = False
        kwargs["use_doc_unwarping"] = False
        kwargs["use_textline_orientation"] = False

    return PaddleOCR(**kwargs)


def _run_predict(ocr: Any, image_path: str) -> list[dict[str, Any]]:
    if hasattr(ocr, "predict"):
        raw = ocr.predict(image_path)
        results: list[dict[str, Any]] = []
        for page in raw or []:
            results.extend(_results_from_v3_page(page))
        return results

    if hasattr(ocr, "ocr"):
        raw = ocr.ocr(image_path)
        return _results_from_v2_raw(raw)

    raise RuntimeError("Installed paddleocr exposes neither predict() nor ocr()")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Run PaddleOCR on an image, output JSON."
    )
    parser.add_argument("image_path", help="Path to the image file")
    parser.add_argument("--lang", default="de", help="Recognition language (default: de)")
    parser.add_argument("--device", default="cpu", help="Inference device (default: cpu)")
    parser.add_argument(
        "--mobile",
        action="store_true",
        help="Force PP-OCRv5 mobile models (recommended on N100/CPU)",
    )
    args = parser.parse_args()

    image_path = Path(args.image_path)
    if not image_path.is_file():
        return _fail(f"Image not found: {args.image_path}")

    try:
        from paddleocr import PaddleOCR  # noqa: F401
    except ImportError:
        return _fail(
            "paddleocr is not installed. Run: python -m pip install paddleocr "
            "(and paddlepaddle CPU wheel — see SKILL.md prerequisites)."
        )

    try:
        ocr = _build_ocr(args.lang, args.device, args.mobile)
        results = _run_predict(ocr, str(image_path))
    except Exception as exc:
        return _fail(f"OCR call failed: {exc}")

    print(
        json.dumps(
            {
                "image": str(image_path),
                "lang": args.lang,
                "results": results,
            },
            ensure_ascii=False,
            indent=2,
        )
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
