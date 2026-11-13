#!/usr/bin/env bash
# E2E: todos os modelos do seletor do Cursor (screenshot 2026-11-12) + tools/passthrough.
set -euo pipefail

HUB_URL="${HUB_URL:-http://127.0.0.1:8080}"
GATEWAY_KEY="${AIHUB_GATEWAY_API_KEY:-}"
REPORT="${REPORT:-/tmp/aihub-e2e-tools-report.tsv}"

if [[ -z "${GATEWAY_KEY}" ]]; then
  echo "AIHUB_GATEWAY_API_KEY is required" >&2
  exit 1
fi

# id|label|body_json_without_model
read -r -d '' MATRIX <<'EOF' || true
aihub/auto|Auto AI-Hub|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"custom","name":"probe_tool"}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
anthropic/claude-haiku-4.5|Claude Haiku 4.5|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"function","name":"probe_tool","parameters":{"type":"object","properties":{}}}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
glm-5.3-flash|GLM 5.3 Flash|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"custom","name":"probe_tool"}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
deepseek-v4.1-flash|DeepSeek V4.1 Flash|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"custom","name":"probe_tool"}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
claude-haiku-5-5|Claude Haiku 5.5|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"custom","name":"probe_tool"}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
gpt-6-luna|6 Luna|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"custom","name":"probe_tool"}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
composer-2.5|Composer 2.5|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"custom","name":"probe_tool"}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
antigravity:claude-opus-4-6-thinking|Claude Opus 4.6 (Thinking)|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"function","name":"probe_tool","parameters":{"type":"object","properties":{}}}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
antigravity:claude-sonnet-4-6|Claude Sonnet 4.6|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"function","name":"probe_tool","parameters":{"type":"object","properties":{}}}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
antigravity:gemini-3.8-flash|Gemini 3.8 Flash|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"function","name":"probe_tool","parameters":{"type":"object","properties":{}}}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
antigravity:gpt-oss-120b-medium|OSS 120B (Medium)|{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"function","name":"probe_tool","parameters":{"type":"object","properties":{}}}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}
EOF

printf 'model\tlabel\thttp\ttool_signal\tresolved_model\tresolved_provider\n' >"$REPORT"
pass=0
fail=0
warn=0

while IFS='|' read -r model_id label body; do
  [[ -z "$model_id" ]] && continue
  payload=$(printf '{"model":"%s",%s' "$model_id" "${body#\{}")
  hdr_file=$(mktemp)
  code=$(curl -sS -o /tmp/aihub-e2e-resp.json -w "%{http_code}" -D "$hdr_file" \
    -H "Authorization: Bearer ${GATEWAY_KEY}" \
    -H "Content-Type: application/json" \
    -d "$payload" \
    "${HUB_URL}/v1/responses") || code="000"
  resolved_model=$(grep -i '^x-aihub-resolved-model:' "$hdr_file" 2>/dev/null | cut -d: -f2- | tr -d ' \r\n' || true)
  resolved_provider=$(grep -i '^x-aihub-resolved-provider:' "$hdr_file" 2>/dev/null | cut -d: -f2- | tr -d ' \r\n' || true)
  rm -f "$hdr_file"

  tool_signal="no"
  if grep -qE 'function_call|probe_tool|"type":"custom"' /tmp/aihub-e2e-resp.json 2>/dev/null; then
    tool_signal="yes"
  fi

  status="FAIL"
  if [[ "$code" == "200" || "$code" == "429" ]]; then
    if [[ "$tool_signal" == "yes" || "$code" == "429" ]]; then
      status="PASS"
      pass=$((pass + 1))
    else
      status="WARN"
      warn=$((warn + 1))
      pass=$((pass + 1))
    fi
  else
    fail=$((fail + 1))
  fi

  printf '%s\n' "$status $label ($model_id) http=$code tool=$tool_signal resolved=${resolved_model:-n/a}/${resolved_provider:-n/a}"
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$model_id" "$label" "$code" "$tool_signal" "$resolved_model" "$resolved_provider" >>"$REPORT"
done <<<"$MATRIX"

echo "--- explicit cursor provider (composer-2.5) ---"
cursor_body='{"instructions":"e2e-stay-intact","tool_choice":"auto","tools":[{"type":"custom","name":"probe_tool"}],"input":[{"type":"input_text","text":"call probe_tool"}],"stream":false}'
cursor_payload=$(printf '{"model":"composer-2.5",%s' "${cursor_body#\{}")
hdr_file=$(mktemp)
cursor_code=$(curl -sS -o /tmp/aihub-e2e-cursor-hdr.json -w "%{http_code}" -D "$hdr_file" \
  -H "Authorization: Bearer ${GATEWAY_KEY}" \
  -H "X-AIHub-Provider: cursor" \
  -H "Content-Type: application/json" \
  -d "$cursor_payload" \
  "${HUB_URL}/v1/responses") || cursor_code="000"
cursor_provider=$(grep -i '^x-aihub-resolved-provider:' "$hdr_file" 2>/dev/null | cut -d: -f2- | tr -d ' \r\n' || true)
rm -f "$hdr_file"
cursor_tool="no"
if grep -qE 'function_call|probe_tool' /tmp/aihub-e2e-cursor-hdr.json 2>/dev/null; then
  cursor_tool="yes"
fi
if [[ "$cursor_code" == "200" && "$cursor_tool" == "yes" ]]; then
  echo "PASS composer-2.5 via X-AIHub-Provider=cursor http=$cursor_code tool=$cursor_tool provider=${cursor_provider:-cursor}"
  pass=$((pass + 1))
elif [[ "$cursor_code" == "200" || "$cursor_code" == "429" ]]; then
  echo "WARN composer-2.5 via X-AIHub-Provider=cursor http=$cursor_code tool=$cursor_tool"
  warn=$((warn + 1))
  pass=$((pass + 1))
else
  echo "FAIL composer-2.5 via X-AIHub-Provider=cursor http=$cursor_code"
  fail=$((fail + 1))
fi

echo "summary pass=$pass warn=$warn fail=$fail report=$REPORT"
[[ "$fail" -eq 0 ]]
