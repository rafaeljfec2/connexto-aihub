import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import type { CursorToolCall } from "@agent-runtime-gateway/compatibility-openai";
import { isRecord } from "@agent-runtime-gateway/compatibility-openai";
import { CLIENT_MCP_SERVER_NAME } from "./constants.js";
import { attachGrammarInput } from "./grammar-input.js";

export function normalizeToolName(name: string): string {
  return name.trim().toLowerCase().replace(/[^a-z0-9]/g, "");
}

function firstString(
  args: Record<string, unknown>,
  ...keys: string[]
): string | undefined {
  for (const key of keys) {
    const value = args[key];
    if (typeof value === "string" && value.trim()) return value.trim();
  }
  return undefined;
}

function mcpPayloadArguments(args: Record<string, unknown>): Record<string, unknown> {
  const nested = firstRecord(args, "args", "arguments", "input", "params", "parameters", "payload");
  return nested ?? args;
}

function firstRecord(
  args: Record<string, unknown>,
  ...keys: string[]
): Record<string, unknown> | undefined {
  for (const key of keys) {
    const value = args[key];
    if (isRecord(value)) return value;
  }
  return undefined;
}

export function resolveInventoryToolName(
  mcpToolName: string,
  clientTools: readonly CursorClientToolSpec[],
): string | null {
  const trimmed = mcpToolName.trim();
  if (!trimmed) return null;
  const normalized = normalizeToolName(trimmed);
  const direct = clientTools.find((tool) => normalizeToolName(tool.name) === normalized);
  if (direct) return direct.name;
  if (normalized.startsWith("client")) {
    const stripped = normalizeToolName(trimmed.replace(/^client[_-]?/i, ""));
    const alias = clientTools.find((tool) => normalizeToolName(tool.name) === stripped);
    if (alias) return alias.name;
  }
  const fallbackNames = new Set(clientTools.map((tool) => tool.name.trim()));
  if (fallbackNames.has(trimmed)) return trimmed;
  return null;
}

export function mcpToolCallToClientTool(
  argsValue: unknown,
  clientTools: readonly CursorClientToolSpec[],
): CursorToolCall | null {
  if (!isRecord(argsValue)) return null;
  const provider = firstString(
    argsValue,
    "providerIdentifier",
    "provider_identifier",
    "provider",
    "server",
    "serverName",
    "server_name",
  );
  if (provider && provider !== CLIENT_MCP_SERVER_NAME) return null;
  const toolName = firstString(argsValue, "toolName", "tool_name", "tool", "name");
  if (!toolName) return null;
  const resolved = resolveInventoryToolName(toolName, clientTools);
  if (!resolved) return null;
  const payload = mcpPayloadArguments(argsValue);
  if (!isRecord(payload)) return null;
  return attachGrammarInput({ name: resolved, arguments: { ...payload } }, clientTools);
}

export function canonicalToolName(name: string): string {
  return normalizeToolName(name);
}
