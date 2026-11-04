#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${1:-18081}"

export AIHUB_SERVER_HOST=127.0.0.1
export AIHUB_SERVER_PORT="$PORT"
export AIHUB_SERVE_EMBEDDED_UI=true
export AIHUB_GATEWAY_API_KEY="${AIHUB_GATEWAY_API_KEY:-01234567890123456789012345678901-e2e-gateway}"
export AIHUB_ADMIN_API_KEY="${AIHUB_ADMIN_API_KEY:-10987654321098765432109876543210-e2e-admin}"
export AIHUB_DATA_DIR="${AIHUB_DATA_DIR:-/tmp/aihub-e2e-passphrase-$$}"
export AIHUB_OPERATOR_PASSPHRASE="${AIHUB_OPERATOR_PASSPHRASE:-e2e-operator-passphrase-value-minimum}"
export AIHUB_UI_SESSION_SECRET="${AIHUB_UI_SESSION_SECRET:-0123456789012345678901234567890123456789012345678901234567890}"

cd "$ROOT"
if [[ -z "${HUB_BIN:-}" ]]; then
  node scripts/sync-frontend-dist.mjs >/dev/null
fi
if [[ -n "${HUB_BIN:-}" && -x "${HUB_BIN}" ]]; then
  exec "${HUB_BIN}"
fi
exec go run ./cmd/ai-hub
