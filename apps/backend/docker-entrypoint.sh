#!/bin/sh
set -eu

if [ -n "${AIHUB_DATABASE_URL:-}" ] && [ "${AIHUB_DOCKER_KEEP_LOOPBACK_DB:-false}" != "true" ]; then
  case "$AIHUB_DATABASE_URL" in
    *"@127.0.0.1:"*|*"@127.0.0.1/"*)
      AIHUB_DATABASE_URL="$(printf '%s' "$AIHUB_DATABASE_URL" | sed 's/@127\.0\.0\.1\([:/]\)/@host.docker.internal\1/')"
      export AIHUB_DATABASE_URL
      ;;
    *"@localhost:"*|*"@localhost/"*)
      AIHUB_DATABASE_URL="$(printf '%s' "$AIHUB_DATABASE_URL" | sed 's/@localhost\([:/]\)/@host.docker.internal\1/')"
      export AIHUB_DATABASE_URL
      ;;
  esac
fi

exec /usr/local/bin/ai-hub "$@"

