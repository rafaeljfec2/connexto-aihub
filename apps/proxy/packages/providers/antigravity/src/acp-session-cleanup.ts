import path from "node:path";
import type { AntigravityAcpClient } from "./acp-client.js";
import type { ListSessionsResult } from "./acp/index.js";

/** Populated in POST /v1/runtime/sessions/clear when namespace is antigravity. */
export interface AntigravityRuntimeSessionClearReport {
  readonly mapped_acp_ids_count: number;
  readonly listed_acp_sessions_count: number;
  readonly acp_sessions_closed: number;
  readonly acp_close_failures: number;
  readonly warnings: readonly string[];
}

const ACP_CONNECT_TIMEOUT_MS = 3_000;
const ACP_LIST_SESSIONS_TIMEOUT_MS = 2_000;
const ACP_CLOSE_SESSION_TIMEOUT_MS = 2_000;

export function resolveWorkingDirectory(
  requestedDirectory: string | undefined,
  defaultDirectory: string,
): string {
  const trimmed = requestedDirectory?.trim() ?? "";
  const candidate = trimmed.length > 0 ? trimmed : defaultDirectory;
  return path.resolve(candidate);
}

export async function closeAntigravityAcpSessionsWithReport(
  mappedAcpSessionIds: readonly string[],
  defaultWorkingDirectory: string,
  clientFactory: (cwd: string) => AntigravityAcpClient,
): Promise<AntigravityRuntimeSessionClearReport> {
  const cwd = resolveWorkingDirectory(undefined, defaultWorkingDirectory);
  const warnings: string[] = [];
  const closedIds = new Set<string>();
  const dedupedMapped = uniqueSessionIds(mappedAcpSessionIds);
  const client = clientFactory(cwd);
  let listedCount = 0;
  let closeFailures = 0;
  try {
    await withTimeout(client.connect(), ACP_CONNECT_TIMEOUT_MS, "connect");
    closeFailures += await closeSessionIds(client, dedupedMapped, closedIds, warnings, "mapped");
    const listed = await readListedSessionIds(client, warnings);
    listedCount = listed.length;
    const remaining = listed.filter((sessionId) => !closedIds.has(sessionId));
    closeFailures += await closeSessionIds(client, remaining, closedIds, warnings, "listed");
  } catch (error) {
    warnings.push(`connect for session cleanup failed: ${errorText(error)}`);
    closeFailures += dedupedMapped.length;
  } finally {
    await safeTerminate(client);
  }
  return {
    mapped_acp_ids_count: dedupedMapped.length,
    listed_acp_sessions_count: listedCount,
    acp_sessions_closed: closedIds.size,
    acp_close_failures: closeFailures,
    warnings,
  };
}

function uniqueSessionIds(sessionIds: readonly string[]): string[] {
  return [...new Set(sessionIds.map((id) => id.trim()).filter((id) => id.length > 0))];
}

async function readListedSessionIds(
  client: AntigravityAcpClient,
  warnings: string[],
): Promise<string[]> {
  let listed: ListSessionsResult;
  try {
    listed = await withTimeout(
      client.listSessions(),
      ACP_LIST_SESSIONS_TIMEOUT_MS,
      "listSessions",
    );
  } catch (error) {
    warnings.push(`listSessions failed: ${errorText(error)}`);
    listed = { sessions: [] };
  }
  return uniqueSessionIds(
    listed.sessions.map((session) =>
      typeof session.sessionId === "string" ? session.sessionId : "",
    ),
  );
}

async function closeSessionIds(
  client: AntigravityAcpClient,
  sessionIds: readonly string[],
  closedIds: Set<string>,
  warnings: string[],
  label: "mapped" | "listed",
): Promise<number> {
  let failures = 0;
  for (const sessionId of sessionIds) {
    const closed = await closeOneSession(client, sessionId, closedIds, warnings, label);
    if (!closed) failures += 1;
  }
  return failures;
}

async function closeOneSession(
  client: AntigravityAcpClient,
  sessionId: string,
  closedIds: Set<string>,
  warnings: string[],
  label: "mapped" | "listed",
): Promise<boolean> {
  try {
    await withTimeout(
      client.closeSession(sessionId),
      ACP_CLOSE_SESSION_TIMEOUT_MS,
      "closeSession",
    );
    closedIds.add(sessionId);
    return true;
  } catch (error) {
    warnings.push(`${label} close failed for ${sessionId}: ${errorText(error)}`);
    return false;
  }
}

function withTimeout<T>(promise: Promise<T>, ms: number, label: string): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(() => {
      reject(new Error(`${label} timed out after ${ms}ms`));
    }, ms);
    promise.then(
      (value) => {
        clearTimeout(timer);
        resolve(value);
      },
      (error: unknown) => {
        clearTimeout(timer);
        reject(error instanceof Error ? error : new Error(String(error)));
      },
    );
  });
}

async function safeTerminate(client: AntigravityAcpClient): Promise<void> {
  try {
    await client.terminate();
  } catch {
    // ignore
  }
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
