#!/usr/bin/env python3
"""Gate 0 Retest: Structured Outputs (json_schema, strict: true) post-catalog-sync.

Runs sequentially, one request per model, no retries, no fallback.
Preserves all prior artifacts by storing outputs under retest-2026-11-10/.
The gateway key is read from .env and never printed.
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
    ("deepseek-v4.1-flash", "opencode-go"),
    ("gpt-6-luna", "opencode-go"),
    ("composer-2.5", "cursor"),
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


def call_model(model: str) -> Tuple[Path, int, float, dict, str]:
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
    raw_str = resp_body.decode("utf-8", errors="replace")

    parsed = {}
    try:
        parsed = json.loads(raw_str)
    except Exception:
        pass

    return out_file, code, elapsed, parsed, raw_str


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
        code_str = err.get("code") or err.get("type") or "quota_429"
        return "BLOCKED", f"HTTP 429 quota block ({code_str})", {"error": err}
    if http_code != 200:
        err = doc.get("error", {})
        msg = str(err.get("message") or doc)
        typ = str(err.get("type") or "")
        low = (msg + " " + typ).lower()
        if any(k in low for k in ["unsupported", "not supported", "json_schema", "structured output", "response_format", "schema"]):
            return "UNSUPPORTED", f"HTTP {http_code} ({typ}): {msg[:120]}", {"error": err}
        return "BLOCKED", f"HTTP {http_code} ({typ}): {msg[:120]}", {"error": err}

    if doc.get("error"):
        err = doc["error"]
        msg = str(err.get("message") or err)
        low = msg.lower()
        if any(k in low for k in ["unsupported", "not supported", "json_schema", "structured output"]):
            return "UNSUPPORTED", f"API error unsupported: {msg[:120]}", {"error": err}
        return "FAIL", f"API error in 200 body: {msg[:120]}", {"error": err}

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
        return "FAIL", f"Parsed JSON is not an object ({type(obj).__name__})", {"parsed": obj}

    keys = set(obj.keys())
    required_keys = {"name", "language", "type"}
    missing = required_keys - keys
    extra = keys - required_keys
    if missing:
        return "FAIL", f"Missing required keys: {sorted(missing)}", {"parsed": obj}
    if extra:
        return "FAIL", f"Additional properties forbidden: {sorted(extra)}", {"parsed": obj}

    non_str = {k: type(v).__name__ for k, v in obj.items() if not isinstance(v, str)}
    if non_str:
        return "FAIL", f"Schema type mismatch (expected string): {non_str}", {"parsed": obj}

    expected_vals = {"name": "Connexto AI-Hub", "language": "Go", "type": "API"}
    val_mismatch = {k: obj[k] for k, exp in expected_vals.items() if obj[k] != exp}
    note = "Strict JSON schema honored without fences or extra properties"
    if val_mismatch:
        note += f" (Note: field values differed from prompt request: {val_mismatch})"

    return "PASS", note, {"parsed": obj, "val_mismatch": val_mismatch}


def get_executions() -> List[dict]:
    req = urllib.request.Request(
        RECORDS_URL,
        headers={"Authorization": f"Bearer {KEY}"},
    )
    with urllib.request.urlopen(req, timeout=15) as resp:
        data = json.loads(resp.read().decode("utf-8"))
        if isinstance(data, list):
            return data
        return data.get("data") or data.get("records") or data.get("items") or []


def main():
    print("=== STARTING GATE 0 POST-SYNC RETEST ===")
    start_utc = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() - 2))
    results = []

    for model, expected_provider in MODELS:
        print(f"\n---> Calling {model} (expected provider: {expected_provider})...")
        out_file, http_code, seconds, doc, raw_str = call_model(model)
        verdict, note, details = evaluate_response(doc, http_code)
        output_txt = extract_output_text(doc)
        usage = doc.get("usage")
        resp_model = doc.get("model")
        resp_status = doc.get("status")

        res_item = {
            "model": model,
            "expected_provider": expected_provider,
            "http_code": http_code,
            "duration_s": round(seconds, 3),
            "response_status": resp_status,
            "response_model": resp_model,
            "verdict": verdict,
            "note": note,
            "output_text": output_txt,
            "details": details,
            "usage": usage,
            "file": str(out_file.relative_to(REPO)),
        }
        results.append(res_item)
        print(f"     Status: HTTP {http_code} in {seconds:.2f}s (status={resp_status}, model={resp_model})")
        print(f"     Verdict: {verdict} ({note})")
        print(f"     Output: {output_txt[:180]!r}")
        print(f"     Usage: {usage}")
        time.sleep(1.0)

    print("\n---> Fetching execution audit records from gateway...")
    all_records = get_executions()
    window_records = [r for r in all_records if (r.get("started_at") or "") >= start_utc]
    print(f"     Total records in window (>= {start_utc}): {len(window_records)}")

    (HERE / "audit-executions-window.json").write_text(json.dumps(window_records, indent=2, ensure_ascii=False))

    matched_records = []
    for item in results:
        m = item["model"]
        matches = [r for r in window_records if r.get("model_id") == m]
        matched = matches[0] if matches else None
        matched_records.append({"model": m, "match_count": len(matches), "record": matched})
        if matched:
            prov = matched.get("provider_id")
            outc = matched.get("outcome")
            up_st = matched.get("upstream_status")
            dur_ms = round((matched.get("duration_ns") or 0) / 1e6, 2)
            rec_usage = matched.get("usage")
            print(f"     Record for {m} (count={len(matches)}): id={matched.get('execution_id')} prov={prov} outcome={outc} upstream_status={up_st} dur={dur_ms}ms usage={rec_usage}")
        else:
            print(f"     WARNING: No execution record found in window for {m}")

    output_summary = {
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "window_start_utc": start_utc,
        "window_record_count": len(window_records),
        "results": results,
        "matched_records": matched_records,
    }

    summary_path = HERE / "retest-summary.json"
    summary_path.write_text(json.dumps(output_summary, indent=2, ensure_ascii=False))
    print(f"\nSaved retest summary to {summary_path.relative_to(REPO)}")


if __name__ == "__main__":
    main()

