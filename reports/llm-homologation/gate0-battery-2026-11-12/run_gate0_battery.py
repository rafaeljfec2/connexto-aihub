#!/usr/bin/env python3
"""Gate 0 homologation battery for the ten Auto MVP allowlist models.

Cases per model (no retries):
  - text: non-stream short completion
  - stream: streaming text
  - json_schema: strict json_schema, non-stream

Reads AIHUB_GATEWAY_API_KEY from repo .env; never prints the key.
"""
from __future__ import annotations

import json
import subprocess
import sys
from dataclasses import dataclass, asdict
from datetime import datetime, timezone
from pathlib import Path

REPO = Path(__file__).resolve().parents[3]
HERE = Path(__file__).resolve().parent
URL = "http://127.0.0.1:8080/v1/responses"

MODELS = [
    "anthropic/claude-haiku-4.5",
    "glm-5.3-flash",
    "deepseek-v4.1-flash",
    "claude-haiku-5-5",
    "gpt-6-luna",
    "composer-2.5",
    "antigravity:gemini-3.8-flash",
    "antigravity:claude-opus-4-6-thinking",
    "antigravity:claude-sonnet-4-6",
    "antigravity:gpt-oss-120b-medium",
]

SCHEMA_PROMPT = (
    "Gere exclusivamente um JSON válido com os campos name, language e type. "
    "Use os valores Connexto AI-Hub, Go e API."
)

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


def load_key() -> str:
    for line in (REPO / ".env").read_text().splitlines():
        if line.startswith("AIHUB_GATEWAY_API_KEY="):
            return line.split("=", 1)[1].strip()
    raise SystemExit("AIHUB_GATEWAY_API_KEY not found")


KEY = load_key()


@dataclass
class CaseResult:
    model: str
    case: str
    http_code: int
    seconds: float
    pass_fail: str
    detail: str


def curl_post(body: dict, timeout: int = 120) -> tuple[int, float, str]:
    raw_path = HERE / "_last_response.json"
    proc = subprocess.run(
        [
            "curl",
            "-sS",
            f"--max-time",
            str(timeout),
            "-o",
            str(raw_path),
            "-w",
            "%{http_code}|%{time_total}",
            "-X",
            "POST",
            URL,
            "-H",
            f"Authorization: Bearer {KEY}",
            "-H",
            "Content-Type: application/json",
            "--data-binary",
            json.dumps(body),
        ],
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        return 0, 0.0, proc.stderr.strip()[:200]
    code_s, secs_s = proc.stdout.strip().split("|", 1)
    try:
        text = raw_path.read_text(encoding="utf-8", errors="replace")
    except OSError as err:
        return int(code_s), float(secs_s), str(err)
    return int(code_s), float(secs_s), text


def judge_text(http_code: int, body: str) -> tuple[str, str]:
    if http_code != 200:
        return "FAIL", f"http_{http_code}"
    try:
        doc = json.loads(body)
    except json.JSONDecodeError as err:
        return "FAIL", f"invalid_json: {err}"
    out = doc.get("output_text") or ""
    if not out and isinstance(doc.get("output"), list):
        for item in doc["output"]:
            for content in item.get("content") or []:
                if isinstance(content, dict) and content.get("text"):
                    out += content["text"]
    if len(out.strip()) < 1:
        return "FAIL", "empty_output"
    return "PASS", out[:80].replace("\n", " ")


def judge_stream(http_code: int, body: str) -> tuple[str, str]:
    if http_code != 200:
        return "FAIL", f"http_{http_code}"
    if "data:" in body or "event:" in body or body.strip().startswith("{"):
        return "PASS", "stream_body_present"
    return "FAIL", "no_stream_markers"


def judge_schema(http_code: int, body: str) -> tuple[str, str]:
    if http_code != 200:
        return "FAIL", f"http_{http_code}"
    try:
        doc = json.loads(body)
    except json.JSONDecodeError as err:
        return "FAIL", f"invalid_json: {err}"
    text = doc.get("output_text") or ""
    if not text and isinstance(doc.get("output"), list):
        for item in doc["output"]:
            for content in item.get("content") or []:
                if isinstance(content, dict) and content.get("text"):
                    text += content["text"]
    import re

    candidate = re.sub(r"^```(?:json)?\s*|\s*```$", "", text.strip())
    try:
        obj = json.loads(candidate)
    except json.JSONDecodeError:
        return "FAIL", "output_not_json"
    if (
        obj.get("name") == "Connexto AI-Hub"
        and obj.get("language") == "Go"
        and obj.get("type") == "API"
    ):
        return "PASS", json.dumps(obj, ensure_ascii=False)
    return "FAIL", json.dumps(obj, ensure_ascii=False)[:120]


def main() -> int:
    results: list[CaseResult] = []
    for model in MODELS:
        cases = [
            (
                "text",
                {
                    "model": model,
                    "input": "Responda somente com a palavra: GATE0_OK",
                    "stream": False,
                    "max_output_tokens": 32,
                },
                judge_text,
                60,
            ),
            (
                "stream",
                {
                    "model": model,
                    "input": "Responda somente com: STREAM_OK",
                    "stream": True,
                    "max_output_tokens": 32,
                },
                judge_stream,
                90,
            ),
            (
                "json_schema",
                {
                    "model": model,
                    "input": SCHEMA_PROMPT,
                    "stream": False,
                    "max_output_tokens": 256,
                    "text": {
                        "format": {
                            "type": "json_schema",
                            "name": "connexto_meta",
                            "strict": True,
                            "schema": SCHEMA,
                        }
                    },
                },
                judge_schema,
                120,
            ),
        ]
        for case_name, body, judge, timeout in cases:
            code, secs, raw = curl_post(body, timeout=timeout)
            pf, detail = judge(code, raw)
            results.append(
                CaseResult(model=model, case=case_name, http_code=code, seconds=secs, pass_fail=pf, detail=detail)
            )
            print(f"{model}\t{case_name}\t{pf}\t{code}\t{secs:.2f}s\t{detail[:60]}", flush=True)

    out = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "results": [asdict(r) for r in results],
    }
    (HERE / "results.json").write_text(json.dumps(out, indent=2), encoding="utf-8")
    failed = sum(1 for r in results if r.pass_fail != "PASS")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
