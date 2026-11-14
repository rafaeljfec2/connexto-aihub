import type { AgentRequestResponse } from "./types.js";

const ALLOW_KINDS = ["allow_always", "allow_once"] as const;

/**
 * Antigravity asks `session/request_permission` before an MCP tool runs.
 * Cancelling that request is recorded as a user denial and the Codex tool never starts.
 */
export function permissionResponse(params: unknown): AgentRequestResponse {
  const optionId = allowOptionId(params);
  if (optionId === undefined) {
    return { result: { outcome: { outcome: "cancelled" } } };
  }
  return { result: { outcome: { outcome: "selected", optionId } } };
}

function allowOptionId(params: unknown): string | undefined {
  if (!isRecord(params) || !Array.isArray(params.options)) return undefined;
  for (const kind of ALLOW_KINDS) {
    for (const option of params.options) {
      if (!isRecord(option) || option.kind !== kind) continue;
      if (typeof option.optionId === "string" && option.optionId.length > 0) {
        return option.optionId;
      }
    }
  }
  return undefined;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
