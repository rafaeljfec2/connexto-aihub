#!/usr/bin/env bash
# Hub runs on the host (Go toolchain); Playwright runs in Jammy (Chromium supported).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${PLAYWRIGHT_IMAGE:-mcr.microsoft.com/playwright:v1.55.0-jammy}"
LOOPBACK_PORT="${AIHUB_E2E_PORT:-18080}"
PASSPHRASE_PORT="${AIHUB_E2E_PASSPHRASE_PORT:-18081}"
HUB_PID=""
PASSPHRASE_PID=""

cleanup() {
  if [[ -n "${HUB_PID}" ]] && kill -0 "${HUB_PID}" 2>/dev/null; then
    kill "${HUB_PID}" 2>/dev/null || true
    wait "${HUB_PID}" 2>/dev/null || true
  fi
  if [[ -n "${PASSPHRASE_PID}" ]] && kill -0 "${PASSPHRASE_PID}" 2>/dev/null; then
    kill "${PASSPHRASE_PID}" 2>/dev/null || true
    wait "${PASSPHRASE_PID}" 2>/dev/null || true
  fi
  fuser -k "${LOOPBACK_PORT}/tcp" "${PASSPHRASE_PORT}/tcp" 2>/dev/null || true
}
trap cleanup EXIT

fuser -k "${LOOPBACK_PORT}/tcp" "${PASSPHRASE_PORT}/tcp" 2>/dev/null || true

cd "$ROOT"
pnpm --filter @connexto/frontend build
node scripts/sync-frontend-dist.mjs
go -C apps/backend build -o "$ROOT/bin/ai-hub-e2e" ./cmd/ai-hub/

export HUB_BIN="$ROOT/bin/ai-hub-e2e"
bash "$ROOT/scripts/e2e-hub.sh" "$LOOPBACK_PORT" &
HUB_PID=$!
bash "$ROOT/scripts/e2e-hub-passphrase.sh" "$PASSPHRASE_PORT" &
PASSPHRASE_PID=$!

for port in "$LOOPBACK_PORT" "$PASSPHRASE_PORT"; do
  ready=0
  for _ in $(seq 1 120); do
    if curl -sf "http://127.0.0.1:${port}/healthz" >/dev/null; then
      ready=1
      break
    fi
    sleep 0.5
  done
  if [[ "${ready}" -ne 1 ]]; then
    echo "FAIL: hub on port ${port} not ready"
    exit 1
  fi
done

docker run --rm --init \
  --add-host=host.docker.internal:host-gateway \
  -v "${ROOT}:${ROOT}" \
  -w "${ROOT}/e2e" \
  -e CI=1 \
  -e AIHUB_E2E_HOST=host.docker.internal \
  -e AIHUB_E2E_PORT="${LOOPBACK_PORT}" \
  -e AIHUB_E2E_PASSPHRASE_PORT="${PASSPHRASE_PORT}" \
  -e AIHUB_E2E_EXTERNAL=1 \
  "${IMAGE}" \
  bash -lc "cd \"${ROOT}/e2e\" && npm install && npx playwright install chromium && npx playwright test"
