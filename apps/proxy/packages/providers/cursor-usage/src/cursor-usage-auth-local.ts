import type { CursorUsageRuntime } from "./cursor-usage-runtime.js";
import {
  isCollectableSessionRecord,
  isUsableStorageState,
} from "./session-validation.js";
import type { CursorUsageAuthOperation } from "./types.js";

export async function refreshLocalBrowserAuthStatus(
  runtime: CursorUsageRuntime,
  operation: CursorUsageAuthOperation,
  sessionId: string,
): Promise<void> {
  const live = runtime.browserPool.getLiveAuth(sessionId);
  if (live === undefined) {
    return;
  }
  const url = live.page.url();
  if (!runtime.collectReadyUrlPattern.test(url)) {
    return;
  }
  if (await runtime.browserPool.pageHasUpstreamChallenge(live.page)) {
    operation.status = "challenge_required";
    return;
  }

  const storageState = await runtime.browserPool.captureStorageState(sessionId);
  const profileReady = runtime.browserPool.hasAuthProfile(operation.accountId);
  if (!isUsableStorageState(storageState) && !profileReady) {
    return;
  }

  try {
    const written = await runtime.sessionStore.setPersisted(
      {
        accountId: operation.accountId,
        status: "authenticated",
        touchedAt: runtime.now(),
        storageState,
      },
      { allowAuthProfileFallback: profileReady },
    );
    const resolved = written
      ? await runtime.sessionStore.resolve(operation.accountId)
      : undefined;
    if (
      !written ||
      resolved === undefined ||
      !isCollectableSessionRecord(resolved, profileReady)
    ) {
      warnSessionPersist(
        operation.accountId,
        written ? "resolve_empty_after_write" : "set_persisted_rejected",
      );
      return;
    }

    operation.status = "authenticated";
    await refreshStorageStateBestEffort(runtime, operation, sessionId);
    await runtime.browserPool.closeAuthSession(sessionId);
  } catch (error) {
    const errno =
      error !== null &&
      typeof error === "object" &&
      "code" in error &&
      typeof (error as NodeJS.ErrnoException).code === "string"
        ? (error as NodeJS.ErrnoException).code
        : undefined;
    warnSessionPersist(
      operation.accountId,
      undefined,
      errno,
      error instanceof Error ? error.message : String(error),
    );
  }
}

async function refreshStorageStateBestEffort(
  runtime: CursorUsageRuntime,
  operation: CursorUsageAuthOperation,
  sessionId: string,
): Promise<void> {
  try {
    const refreshedState =
      await runtime.browserPool.captureStorageState(sessionId);
    if (!isUsableStorageState(refreshedState)) {
      return;
    }
    await runtime.sessionStore.setPersisted({
      accountId: operation.accountId,
      status: "authenticated",
      touchedAt: runtime.now(),
      storageState: refreshedState,
    });
  } catch {
    // Best-effort refresh before closing headed browser.
  }
}

function warnSessionPersist(
  accountId: string,
  reason?: string,
  errno?: string,
  message?: string,
): void {
  console.warn(
    JSON.stringify({
      level: "warn",
      msg: "session_persist_failed",
      accountId,
      reason,
      errno,
      message,
    }),
  );
}
