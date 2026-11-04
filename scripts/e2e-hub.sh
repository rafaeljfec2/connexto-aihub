#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${1:-18080}"

export AIHUB_SERVER_HOST=127.0.0.1
export AIHUB_SERVER_PORT="$PORT"
export AIHUB_SERVE_EMBEDDED_UI=true
export AIHUB_GATEWAY_API_KEY="${AIHUB_GATEWAY_API_KEY:-01234567890123456789012345678901-e2e-gateway}"
export AIHUB_ADMIN_API_KEY="${AIHUB_ADMIN_API_KEY:-10987654321098765432109876543210-e2e-admin}"
export AIHUB_DATA_DIR="${AIHUB_DATA_DIR:-/tmp/aihub-e2e-data}"

cd "$ROOT"
node scripts/sync-frontend-dist.mjs >/dev/null
exec go run ./cmd/ai-hub
