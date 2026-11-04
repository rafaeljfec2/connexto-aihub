#!/usr/bin/env bash
# Embedded production smoke — waits for hub readiness before HTTP checks.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${1:-18080}"
BASE="http://127.0.0.1:${PORT}"
HUB_BIN="${HUB_BIN:-$ROOT/bin/ai-hub-smoke}"
LOG_FILE="$(mktemp)"
HUB_PID=""

cleanup() {
  if [[ -n "${HUB_PID}" ]] && kill -0 "${HUB_PID}" 2>/dev/null; then
    kill "${HUB_PID}" 2>/dev/null || true
    wait "${HUB_PID}" 2>/dev/null || true
  fi
  rm -f "${LOG_FILE}"
}
trap cleanup EXIT

export AIHUB_SERVER_HOST=127.0.0.1
export AIHUB_SERVER_PORT="$PORT"
export AIHUB_SERVE_EMBEDDED_UI=true
export AIHUB_GATEWAY_API_KEY="${AIHUB_GATEWAY_API_KEY:-01234567890123456789012345678901-e2e-gateway}"
export AIHUB_ADMIN_API_KEY="${AIHUB_ADMIN_API_KEY:-10987654321098765432109876543210-e2e-admin}"
export AIHUB_DATA_DIR="${AIHUB_DATA_DIR:-/tmp/aihub-embedded-smoke-$$}"

cd "$ROOT"
node scripts/sync-frontend-dist.mjs >/dev/null
go build -o "$HUB_BIN" ./cmd/ai-hub/

echo "Starting hub on ${BASE} (log: ${LOG_FILE})"
"$HUB_BIN" >"${LOG_FILE}" 2>&1 &
HUB_PID=$!

ready=0
for _ in $(seq 1 60); do
  if curl -sf --max-time 2 "${BASE}/healthz" >/dev/null; then
    ready=1
    break
  fi
  if ! kill -0 "${HUB_PID}" 2>/dev/null; then
    echo "FAIL: hub process exited before readiness"
    cat "${LOG_FILE}"
    exit 1
  fi
  sleep 0.5
done

if [[ "${ready}" -ne 1 ]]; then
  echo "FAIL: readiness timeout (60s)"
  cat "${LOG_FILE}"
  exit 1
fi

if ! grep -q "listening" "${LOG_FILE}" 2>/dev/null; then
  echo "WARN: no 'listening' line in hub log yet (continuing after /healthz 200)"
fi

curl_check() {
  local label="$1"
  shift
  local code
  code="$(curl -sS -o /tmp/aihub-smoke-body -w '%{http_code}' "$@" 2>/dev/null)"
  if [[ -z "${code}" ]]; then
    code="000"
  fi
  if [[ "${code}" == "000" ]] || [[ -z "${code}" ]]; then
    echo "FAIL: ${label} — HTTP ${code:-empty}"
    return 1
  fi
  echo "PASS: ${label} — HTTP ${code}"
}

fail=0
curl_check "GET /healthz" -f "${BASE}/healthz" || fail=1
curl_check "GET / (index.html)" "${BASE}/" || fail=1
curl_check "GET /executions SPA route" "${BASE}/executions" || fail=1
curl_check "GET /api/ui/session" "${BASE}/api/ui/session" || fail=1

session_body="$(cat /tmp/aihub-smoke-body)"
if [[ "${session_body}" != *'"authenticated":true'* ]] || [[ "${session_body}" != *'loopback_trust'* ]]; then
  echo "FAIL: session body unexpected: ${session_body}"
  fail=1
else
  echo "PASS: session loopback_trust payload"
fi

curl_check "GET /v1/records/executions (gateway)" "${BASE}/v1/records/executions?limit=1" || fail=1
curl_check "GET /api/resource-policies (admin)" "${BASE}/api/resource-policies" || fail=1

unauth_code="$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer invalid-token" "${BASE}/api/resource-policies" 2>&1)" || unauth_code="000"
if [[ "${unauth_code}" == "401" ]] || [[ "${unauth_code}" == "403" ]]; then
  echo "PASS: invalid bearer rejected — HTTP ${unauth_code}"
else
  echo "FAIL: invalid bearer — HTTP ${unauth_code}"
  fail=1
fi

if [[ "${fail}" -ne 0 ]]; then
  echo "--- hub log ---"
  cat "${LOG_FILE}"
  exit 1
fi

echo "All embedded production smoke checks passed."
