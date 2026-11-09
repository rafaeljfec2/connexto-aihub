#!/usr/bin/env python3
"""Phase 3: JSON contract classification per model.

Three independent variants per model, one request each, no retries:
  - prompt_only:            JSON requested only by instruction text
  - text_format_json_object: Responses API text.format = json_object
  - text_format_json_schema: Responses API text.format = json_schema (strict)

The gateway key is read from the repository .env and never printed.
"""
import json
import re
import subprocess
from pathlib import Path
from typing import Tuple

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[2]
URL = "http://127.0.0.1:8080/v1/responses"
MODELS = ["anthropic/claude-haiku-4.5", "glm-5.3-flash", "claude-haiku-5-5"]
PROMPT = ("Gere exclusivamente um JSON válido com os campos name, language e type. "
          "Use os valores Connexto AI-Hub, Go e API.")
SCHEMA = {
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "language": {"type": "string"},
        "type": {"type": "string"},
    },
    "required": ["name", "language", "type"],
    "additionalProperties": False,
}
VARIANTS = {
    "prompt_only": None,
    "text_format_json_object": {"format": {"type": "json_object"}},
    "text_format_json_schema": {
        "format": {"type": "json_schema", "name": "connexto_meta", "strict": True, "schema": SCHEMA}
    },
}


def load_key() -> str:
    for line in (REPO / ".env").read_text().splitlines():
        if line.startswith("AIHUB_GATEWAY_API_KEY="):
            return line.split("=", 1)[1].strip()
    raise SystemExit("AIHUB_GATEWAY_API_KEY not found")


KEY = load_key()


def call(model: str, variant: str) -> Tuple[Path, int, float]:
    body = {"model": model, "input": PROMPT, "stream": False, "max_output_tokens": 256}
    if VARIANTS[variant]:
        body["text"] = VARIANTS[variant]
    out = HERE / f"{re.sub(r'[/:]', '-', model)}-{variant}.json"
    proc = subprocess.run(
        ["curl", "-sS", "--max-time", "60", "-o", str(out), "-w", "%{http_code}|%{time_total}",
         "-X", "POST", URL, "-H", f"Authorization: Bearer {KEY}",
         "-H", "Content-Type: application/json", "--data-binary", json.dumps(body)],
        capture_output=True, text=True,
    )
    code, seconds = proc.stdout.strip().split("|")
    return out, int(code), float(seconds)


def text_of(doc: dict) -> str:
    if isinstance(doc.get("output_text"), str):
        return doc["output_text"]
    parts = []
    for item in doc.get("output", []) or []:
        for content in item.get("content", []) or []:
            if isinstance(content, dict):
                parts.append(content.get("text", ""))
    return "".join(parts)


def judge(text: str) -> Tuple[str, str]:
    stripped = text.strip()
    fenced = stripped.startswith("```")
    candidate = re.sub(r"^```(?:json)?\s*|\s*```$", "", stripped)
    try:
        obj = json.loads(candidate)
    except json.JSONDecodeError as error:
        return "NOT_JSON", str(error)[:80]
    if not isinstance(obj, dict):
        return "NOT_OBJECT", str(obj)[:80]
    valid = (obj.get("name") == "Connexto AI-Hub" and obj.get("language") == "Go"
             and obj.get("type") == "API")
    if not valid:
        return "WRONG_VALUES", json.dumps(obj, ensure_ascii=False)[:120]
    return ("PASS_FENCED" if fenced else "PASS"), ""


def main() -> None:
    summary = []
    for model in MODELS:
        for variant in VARIANTS:
            out, code, seconds = call(model, variant)
            try:
                doc = json.loads(out.read_text())
            except (json.JSONDecodeError, FileNotFoundError):
                summary.append({"model": model, "variant": variant, "http": code,
                                "seconds": seconds, "verdict": "NO_JSON_RESPONSE", "note": ""})
                continue
            if doc.get("error"):
                summary.append({"model": model, "variant": variant, "http": code,
                                "seconds": seconds, "verdict": "API_ERROR",
                                "note": json.dumps(doc["error"], ensure_ascii=False)[:160]})
                continue
            verdict, note = judge(text_of(doc))
            summary.append({"model": model, "variant": variant, "http": code,
                            "seconds": seconds, "verdict": verdict, "note": note})
            print(f"{model:30s} {variant:26s} http={code} {seconds:.2f}s {verdict} {note}", flush=True)
    (HERE / "summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=1))
    print("summary.json written:", len(summary), "entries")


if __name__ == "__main__":
    main()
