import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { CLIENT_MCP_SERVER_NAME } from "./constants.js";
import { clientMcpToolDefinitions } from "./definitions.js";

export const CLIENT_MCP_TOOLS_ENV = "CURSOR_SDK_BRIDGE_CLIENT_TOOLS_JSON";
export const CLIENT_MCP_CALLBACK_URL_ENV = "CURSOR_SDK_BRIDGE_CALLBACK_URL";
export const CLIENT_MCP_CALLBACK_TOKEN_ENV = "CURSOR_SDK_BRIDGE_CALLBACK_TOKEN";

const require = createRequire(import.meta.url);

export interface ClientMcpForwardingConfig {
  readonly callbackUrl: string;
  readonly callbackToken: string;
}

export function clientForwardingMcpServers(
  clientTools: readonly CursorClientToolSpec[],
  config: ClientMcpForwardingConfig,
): Record<string, unknown> {
  const entryPath = resolveMcpServerEntryPath();
  const command = process.execPath;
  const args = buildMcpServerExecArgs(entryPath);
  return {
    [CLIENT_MCP_SERVER_NAME]: {
      type: "stdio",
      command,
      args,
      env: {
        [CLIENT_MCP_CALLBACK_URL_ENV]: config.callbackUrl,
        [CLIENT_MCP_CALLBACK_TOKEN_ENV]: config.callbackToken,
        [CLIENT_MCP_TOOLS_ENV]: JSON.stringify(clientMcpToolDefinitions(clientTools)),
      },
    },
  };
}

function resolveMcpServerEntryPath(): string {
  return fileURLToPath(new URL("./stdio-entry.ts", import.meta.url));
}

function resolveTsxLoaderPath(): string {
  return require.resolve("tsx/esm");
}

function buildMcpServerExecArgs(entryPath: string): string[] {
  // Absolute tsx loader so the child resolves even when cwd lacks node_modules/tsx.
  return ["--import", resolveTsxLoaderPath(), entryPath];
}
