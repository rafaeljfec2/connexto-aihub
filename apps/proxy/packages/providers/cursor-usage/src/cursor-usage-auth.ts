import crypto from "node:crypto";
import { BridgeSessionNotFoundError } from "./host-bridge-client.js";
import { BrowserUnavailableError } from "./playwright-browser.js";
import { refreshLocalBrowserAuthStatus } from "./cursor-usage-auth-local.js";
import type { CursorUsageRuntime } from "./cursor-usage-runtime.js";
import {
  isCollectableSessionRecord,
} from "./session-validation.js";
import type {
  CursorUsageAuthOperation,
  CursorUsageAuthStartResult,
  CursorUsageAuthStatusResult,
  CursorUsageCollectSessionStatus,
} from "./types.js";

export async function startCursorUsageAuth(
  runtime: CursorUsageRuntime,
  accountId: string,
): Promise<CursorUsageAuthStartResult> {
  return runtime.locks.run(accountId, async () => {
    purgeExpiredAuthOperations(runtime);
    const sessionId: string = crypto.randomUUID();
    const expiresAt = runtime.now() + runtime.authOperationTtlMs;
    let effectiveSessionId: string = sessionId;
    let isAlreadyAuthenticated = false;

    try {
      if (runtime.hostBridgeClient) {
        const launched = await runtime.hostBridgeClient.createSession({
          sessionId,
          principalId: accountId,
          target: "cursor",
          targetUrl: runtime.loginUrl,
          headless: false,
        });
        if (launched?.sessionId) {
          effectiveSessionId = launched.sessionId;
        }
        if (launched?.status === "authenticated") {
          isAlreadyAuthenticated = true;
        }
      } else {
        await runtime.browserPool.openAuthSession(
          sessionId,
          accountId,
          runtime.loginUrl,
        );
      }
    } catch (error) {
      if (error instanceof BrowserUnavailableError) {
        throw error;
      }
      throw new BrowserUnavailableError(error);
    }

    const interactionUrl = `${runtime.interactionUrlBase}/v1/providers/cursor/auth/${effectiveSessionId}/interaction`;
    const operation: CursorUsageAuthOperation = {
      sessionId: effectiveSessionId,
      accountId,
      status: isAlreadyAuthenticated
        ? "authenticated"
        : "authentication_pending",
      createdAt: runtime.now(),
      expiresAt,
      interactionUrl,
    };
    runtime.authOperations.set(effectiveSessionId, operation);

    if (isAlreadyAuthenticated) {
      await runtime.sessionStore
        .setPersisted(
          {
            accountId,
            status: "authenticated",
            touchedAt: runtime.now(),
            authProfileBacked: true,
          },
          { allowAuthProfileFallback: true },
        )
        .catch(() => undefined);
    }

    return {
      status: "authentication_pending",
      sessionId: effectiveSessionId,
      expiresAt: new Date(expiresAt).toISOString(),
      interactionUrl,
      browserMode: runtime.hostBridgeClient
        ? "headed"
        : runtime.authHeadless
          ? "headless"
          : "headed",
    };
  });
}

export async function getCursorUsageAuthStatus(
  runtime: CursorUsageRuntime,
  sessionId: string,
): Promise<CursorUsageAuthStatusResult> {
  purgeExpiredAuthOperations(runtime);
  const operation = runtime.authOperations.get(sessionId);
  if (operation === undefined) {
    return {
      sessionId,
      accountId: "",
      status: "expired",
      expiresAt: new Date(0).toISOString(),
    };
  }

  if (operation.status === "authentication_pending") {
    await refreshPendingAuthStatus(runtime, operation, sessionId);
  }

  return {
    sessionId,
    accountId: operation.accountId,
    status: operation.status,
    expiresAt: new Date(operation.expiresAt).toISOString(),
  };
}

export async function getCursorCollectSessionStatus(
  runtime: CursorUsageRuntime,
  accountId: string,
): Promise<CursorUsageCollectSessionStatus> {
  const session = await runtime.sessionStore.resolve(accountId);
  const profilePresent = runtime.browserPool.hasAuthProfile(accountId);
  if (
    session === undefined ||
    session.status !== "authenticated" ||
    !isCollectableSessionRecord(session, profilePresent)
  ) {
    if (profilePresent) {
      return {
        status: "authenticated",
        accountId,
        touchedAt: new Date(runtime.now()).toISOString(),
      };
    }
    return { status: "authentication_required", accountId };
  }
  return {
    status: "authenticated",
    accountId,
    touchedAt: new Date(session.touchedAt).toISOString(),
  };
}

export function getCursorInteractionPage(
  runtime: CursorUsageRuntime,
  sessionId: string,
): { url: string } | undefined {
  const live = runtime.browserPool.getLiveAuth(sessionId);
  if (live === undefined) {
    return undefined;
  }
  return { url: live.page.url() };
}

export async function cancelCursorUsageAuth(
  runtime: CursorUsageRuntime,
  sessionId: string,
): Promise<void> {
  const operation = runtime.authOperations.get(sessionId);
  if (runtime.hostBridgeClient && operation) {
    try {
      await runtime.hostBridgeClient.closeSession(sessionId, operation.accountId);
    } catch {
      // Best-effort remote bridge cleanup
    }
  }
  if (operation !== undefined) {
    runtime.authOperations.delete(sessionId);
  }
  await runtime.browserPool.closeAuthSession(sessionId);
}

export function purgeExpiredAuthOperations(runtime: CursorUsageRuntime): void {
  const now = runtime.now();
  for (const [sessionId, op] of runtime.authOperations) {
    if (now > op.expiresAt) {
      runtime.authOperations.delete(sessionId);
      void runtime.browserPool.closeAuthSession(sessionId);
    }
  }
}

async function refreshPendingAuthStatus(
  runtime: CursorUsageRuntime,
  operation: CursorUsageAuthOperation,
  sessionId: string,
): Promise<void> {
  if (runtime.hostBridgeClient) {
    try {
      const remote = await runtime.hostBridgeClient.getSession(
        sessionId,
        operation.accountId,
      );
      if (remote.status === "authenticated") {
        operation.status = "authenticated";
        await runtime.sessionStore.setPersisted(
          {
            accountId: operation.accountId,
            status: "authenticated",
            touchedAt: runtime.now(),
            authProfileBacked: true,
          },
          { allowAuthProfileFallback: true },
        );
      } else if (remote.status === "failed") {
        operation.status = "authentication_failed";
      } else if (remote.status === "expired") {
        operation.status = "expired";
      }
      return;
    } catch (bridgeErr) {
      if (bridgeErr instanceof BridgeSessionNotFoundError) {
        operation.status = "expired";
      }
    }
  }

  await refreshLocalBrowserAuthStatus(runtime, operation, sessionId);
}
