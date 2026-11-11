#!/usr/bin/env python3
"""Gate 0 Post-Fix Retest: Structured Outputs (json_schema, strict: true).

Targets:
- composer-2.5 (cursor) — primary fix verification
- deepseek-v4.1-flash (opencode-go) — retest after transient 429 overload
- antigravity:gemini-3.8-flash (antigravity) — regression check for proxy change

Sequential, one call per model, no retries, no fallback.
"""
import json
import re
import time
import urllib.request
from pathlib import Path
from typing import List, Tuple

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]
URL = "http://127.0.0.1:8080/v1/responses"
RECORDS_URL = "http://127.0.0.1:8080/v1/records/executions?limit=20"

MODELS = [
    ("composer-2.5", "cursor"),
    ("deepseek-v4.1-flash", "opencode-go"),
    ("antigravity:gemini-3.8-flash", "antigravity"),
]

PROMPT = (
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
PAYLOAD_TEXT_FORMAT = {
    "type": "json_schema",
    "name": "connexto_meta",
    "strict": True,
    "schema": SCHEMA,
}


def load_key() -> str:
    for line in (REPO / ".env").read_text().splitlines():
        if line.startswith("AIHUB_GATEWAY_API_KEY="):
            return line.split("=", 1)[1].strip()
    raise SystemExit("AIHUB_GATEWAY_API_KEY not found in .env")


KEY = load_key()


def call_model(model: str) -> Tuple[Path, int, float, dict]:
    body = {
        "model": model,
        "input": PROMPT,
        "stream": False,
        "max_output_tokens": 256,
        "text": {"format": PAYLOAD_TEXT_FORMAT},
    }
    slug = re.sub(r"[/:]", "-", model)
    out_file = HERE / f"{slug}-text_format_json_schema.json"

    t0 = time.time()
    req = urllib.request.Request(
        URL,
        data=json.dumps(body).encode("utf-8"),
        headers={
            "Authorization": f"Bearer {KEY}",
            "Content-Type": "application/json",
        },
        method="POST",
    )

    code = 0
    resp_body = b""
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            code = resp.status
            resp_body = resp.read()
    except urllib.error.HTTPError as err:
        code = err.code
        resp_body = err.read()
    except Exception as exc:
        code = 999
        resp_body = json.dumps({"error": {"type": "client_error", "message": str(exc)}}).encode("utf-8")

    elapsed = time.time() - t0
    out_file.write_bytes(resp_body)

    parsed = {}
    try:
        parsed = json.loads(resp_body.decode("utf-8", errors="replace"))
    except Exception:
        pass

    return out_file, code, elapsed, parsed


def extract_output_text(doc: dict) -> str:
    if isinstance(doc.get("output_text"), str):
        return doc["output_text"]
    parts = []
    for item in doc.get("output", []) or []:
        for content in item.get("content", []) or []:
            if isinstance(content, dict):
                parts.append(content.get("text", ""))
    return "".join(parts)


def evaluate_response(doc: dict, http_code: int) -> Tuple[str, str, dict]:
    if http_code == 429:
        err = doc.get("error", {})
        return "BLOCKED", f"HTTP 429 ({err.get('type') or 'quota'}): {str(err.get('message', ''))[:100]}", {"error": err}
    if http_code != 200:
        err = doc.get("error", {})
        msg = str(err.get("message") or doc)
        typ = str(err.get("type") or "")
        return "FAIL" if typ == "upstream_invalid_structured_output" else "BLOCKED", f"HTTP {http_code} ({typ}): {msg[:120]}", {"error": err}

    if doc.get("error"):
        return "FAIL", f"API error in 200 body: {doc['error']}", {"error": doc["error"]}
    if doc.get("status") and doc.get("status") != "completed":
        return "FAIL", f"Response status is {doc.get('status')}", {"status": doc.get("status")}

    text = extract_output_text(doc)
    stripped = text.strip()
    if not stripped:
        return "FAIL", "Empty output text", {"raw_text": text}
    if "```" in stripped:
        return "FAIL", "Markdown fences present in output", {"raw_text": text}
    if not (stripped.startswith("{") and stripped.endswith("}")):
        return "FAIL", "Extra text outside JSON object", {"raw_text": text}

    try:
        obj = json.loads(stripped)
    except json.JSONDecodeError as err:
        return "FAIL", f"Invalid JSON syntax: {err}", {"raw_text": text}

    if not isinstance(obj, dict):
        return "FAIL", "Parsed JSON is not an object", {"parsed": obj}

    keys = set(obj.keys())
    required = {"name", "language", "type"}
    if required - keys:
        return "FAIL", f"Missing required keys: {sorted(required - keys)}", {"parsed": obj}
    if keys - required:
        return "FAIL", f"Extra keys: {sorted(keys - required)}", {"parsed": obj}
    if not all(isinstance(obj[k], str) for k in required):
        return "FAIL", "Non-string value in fields", {"parsed": obj}

    return "PASS", "Strict JSON schema honored without fences or extra properties", {"parsed": obj}


def get_executions() -> List[dict]:
    req = urllib.request.Request(RECORDS_URL, headers={"Authorization": f"Bearer {KEY}"})
    with urllib.request.urlopen(req, timeout=15) as resp:
        data = json.loads(resp.read().decode("utf-8"))
        return data if isinstance(data, list) else (data.get("data") or data.get("records") or [])


def main() -> None:
    print("=== STARTING GATE 0 POST-FIX RETEST ===")
    start_utc = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - 2))
    results = []

    for model, expected_provider in MODELS:
        print(f"\n---> Calling {model} (expected provider: {expected_provider})...")
        out_file, http_code, seconds, doc = call_model(model)
        verdict, note, details = evaluate_response(doc, http_code)
        output_txt = extract_output_text(doc)
        usage = doc.get("usage")
        results.append({
            "model": model,
            "expected_provider": expected_provider,
            "http_code": http_code,
            "duration_s": round(seconds, 3),
            "response_status": doc.get("status"),
            "response_model": doc.get("model"),
            "verdict": verdict,
            "note": note,
            "output_text": output_txt,
            "details": details,
            "usage": usage,
            "file": str(out_file.relative_to(REPO)),
        })
        print(f"     HTTP {http_code} in {seconds:.2f}s | {verdict} ({note})")
        print(f"     Output: {output_txt!r}")
        time.sleep(1.0)

    print("\n---> Auditing execution records...")
    all_records = get_executions()
    window_records = [r for r in all_records if (r.get("started_at") or "") >= start_utc]
    (HERE / "audit-executions-window.json").write_text(json.dumps(window_records, indent=2, ensure_ascii=False))
    print(f"     Records in window (>= {start_utc}): {len(window_records)}")

    matched_records = []
    for item in results:
        matches = [r for r in window_records if r.get("model_id") == item["model"]]
        matched = matches[0] if matches else None
        matched_records.append({"model": item["model"], "match_count": len(matches), "record": matched})
        if matched:
            print(f"     {item['model']}: id={matched.get('execution_id')} prov={matched.get('provider_id')} outcome={matched.get('outcome')} upstream_status={matched.get('upstream_status')} dur={round((matched.get('duration_ns') or 0)/1e6, 2)}ms usage={matched.get('usage')}")

    summary = {
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "window_start_utc": start_utc,
        "window_record_count": len(window_records),
        "results": results,
        "matched_records": matched_records,
    }
    (HERE / "retest-summary.json").write_text(json.dumps(summary, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()

