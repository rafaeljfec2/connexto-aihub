import {
  CLIENT_MCP_CALLBACK_TOKEN_ENV,
  CLIENT_MCP_CALLBACK_URL_ENV,
  CLIENT_MCP_TOOLS_ENV,
} from "./forwarding.js";
import type { ClientMcpToolDefinition } from "./definitions.js";
import { runClientForwardingMcpServer } from "./server.js";

function loadTools(): ClientMcpToolDefinition[] {
  const raw = process.env[CLIENT_MCP_TOOLS_ENV];
  if (!raw?.trim()) return [];
  try {
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as ClientMcpToolDefinition[]) : [];
  } catch {
    return [];
  }
}

await runClientForwardingMcpServer({
  tools: loadTools(),
  callbackUrl: process.env[CLIENT_MCP_CALLBACK_URL_ENV] ?? "",
  callbackToken: process.env[CLIENT_MCP_CALLBACK_TOKEN_ENV] ?? "",
});
