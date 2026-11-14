import fs from "node:fs";
import {
  AcpClient,
  AGY_CLI_PATH_ENV,
  DEFAULT_AGY_BINARY,
  defaultAgyPathCandidates,
} from "./acp/index.js";
import { DEFAULT_AUTH_METHOD_ID, DEFAULT_PROCESS_GRACE_MS } from "./acp/constants.js";
import type {
  AgentTurnResult,
  ContentBlock,
  ListSessionsResult,
  RunAgentTurnOptions,
  SessionConfigOption,
} from "./acp/index.js";

export interface AntigravityAcpClient {
  connect(): Promise<void>;
  authenticate?(methodId?: string): Promise<void>;
  createSession(cwd: string, mcpServers?: unknown[]): Promise<{ sessionId: string }>;
  resumeSession(
    sessionId: string,
    cwd: string,
    mcpServers?: unknown[],
  ): Promise<Record<string, unknown>>;
  listSessions(): Promise<ListSessionsResult>;
  setSessionConfig(
    sessionId: string,
    configId: string,
    value: string | boolean,
  ): Promise<readonly SessionConfigOption[]>;
  prompt(
    sessionId: string,
    blocks: readonly ContentBlock[],
    options?: RunAgentTurnOptions,
  ): Promise<AgentTurnResult>;
  cancelTurn(sessionId: string): void;
  closeSession(sessionId: string): Promise<void>;
  terminate(): Promise<void>;
}

const DEFAULT_CLIENT_INFO = {
  name: "agent-runtime-gateway",
  version: "0.0.0",
} as const;

export function createDefaultClient(cwd: string, homeDir?: string): AntigravityAcpClient {
  let env: Record<string, string> | undefined;
  if (homeDir) {
    env = {
      ...(process.env as Record<string, string>),
      HOME: homeDir,
      ANTIGRAVITY_APP_DATA_DIR: homeDir,
      DISPLAY: process.env["DISPLAY"] ?? ":0",
    };
  }
  const safeCwd = fs.existsSync(cwd) ? cwd : process.cwd();
  return new AcpClient({
    command: resolveAgyCommand(),
    cwd: safeCwd,
    env,
    clientInfo: DEFAULT_CLIENT_INFO,
    authMethodId: DEFAULT_AUTH_METHOD_ID,
    timeouts: { processGraceMs: DEFAULT_PROCESS_GRACE_MS },
  });
}

export async function terminatePreservingCause(
  client: AntigravityAcpClient,
  cause: unknown,
): Promise<void> {
  try {
    await client.terminate();
  } catch (cleanupError) {
    throw new AggregateError(
      [cause, cleanupError],
      "Antigravity setup and ACP process cleanup failed.",
    );
  }
}

function resolveAgyCommand(): string {
  const configuredPath = process.env[AGY_CLI_PATH_ENV]?.trim();
  if (configuredPath) return configuredPath;
  const homeBinary = defaultAgyPathCandidates().find((candidate) => fs.existsSync(candidate));
  return homeBinary ?? DEFAULT_AGY_BINARY;
}
