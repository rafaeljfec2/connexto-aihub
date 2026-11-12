import { buildHostBridgeUsageCollectScript } from "./bridge-usage-collect.js";
import {
  BridgeSessionNotFoundError,
  type HostBrowserBridgeClient,
} from "./host-bridge-client.js";
import type { CursorUsageRuntime } from "./cursor-usage-runtime.js";
import type { CursorUsageCollectResult } from "./types.js";
import { normalizeCursorUsagePayload } from "./usage-normalize.js";

type EvaluateResult = {
  readonly ok?: boolean;
  readonly data?: unknown;
  readonly error?: string;
};

const COLLECT_DASHBOARD_TARGET = "https://grok.com/dashboard";
const SESSION_WARMUP_MS = 2_500;
const EVALUATE_RETRY_DELAY_MS = 1_000;
const EVALUATE_MAX_ATTEMPTS = 4;

export async function collectUsageViaBridge(
  runtime: CursorUsageRuntime,
  accountId: string,
): Promise<CursorUsageCollectResult> {
  const bridge = runtime.hostBridgeClient;
  if (!bridge) {
    return { status: "collection_failed", provider: "cursor", accountId };
  }

  let createdForCollect = false;
  let activeSessionId: string | undefined;

  try {
    const active = await resolveBridgeCollectSession(bridge, accountId);
    if (active.kind === "auth_required") {
      return { status: "authentication_required", provider: "cursor", accountId };
    }
    if (active.kind === "auth_pending") {
      return { status: "authentication_pending", provider: "cursor", accountId };
    }

    createdForCollect = active.createdForCollect;
    activeSessionId = active.sessionId;

    const evalScript = buildHostBridgeUsageCollectScript();
    let evalResult = await evaluateUsageWithRetry(
      bridge,
      active.sessionId,
      accountId,
      evalScript,
    );

    if ((!evalResult?.ok || !evalResult.data) && !createdForCollect) {
      const launched = await bridge.createSession({
        principalId: accountId,
        target: "cursor",
        targetUrl: COLLECT_DASHBOARD_TARGET,
        headless: true,
      });
      createdForCollect = true;
      activeSessionId = launched.sessionId;
      await sleep(SESSION_WARMUP_MS);
      evalResult = await evaluateUsageWithRetry(
        bridge,
        launched.sessionId,
        accountId,
        evalScript,
      );
    }

    if (evalResult?.ok && evalResult.data) {
      const usage = normalizeCursorUsagePayload(evalResult.data);
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

      return {
        status: "available",
        provider: "cursor",
        accountId,
        collectedAt: new Date(runtime.now()).toISOString(),
        usage,
        metadata: {
          source: "cursor_dashboard",
          collectionMethod: "host_browser_bridge",
        },
      };
    }

    return { status: "collection_failed", provider: "cursor", accountId };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    if (/timeout/i.test(message)) {
      return { status: "timeout", provider: "cursor", accountId };
    }
    return { status: "collection_failed", provider: "cursor", accountId };
  } finally {
    if (createdForCollect && activeSessionId) {
      await bridge
        .closeSession(activeSessionId, accountId)
        .catch(() => undefined);
    }
  }
}

type BridgeSessionResolution =
  | { readonly kind: "auth_required" }
  | { readonly kind: "auth_pending" }
  | {
      readonly kind: "ready";
      readonly sessionId: string;
      readonly createdForCollect: boolean;
    };

async function resolveBridgeCollectSession(
  bridge: HostBrowserBridgeClient,
  accountId: string,
): Promise<BridgeSessionResolution> {
  const sessions = await bridge.listSessions(accountId);
  let active = sessions.find(
    (session) =>
      session.connected !== false &&
      (session.status === "authenticated" ||
        session.status === "waiting_for_login"),
  );

  let createdForCollect = false;
  if (!active) {
    try {
      active = await bridge.createSession({
        principalId: accountId,
        target: "cursor",
        targetUrl: COLLECT_DASHBOARD_TARGET,
        headless: true,
      });
      createdForCollect = true;
      await sleep(SESSION_WARMUP_MS);
    } catch {
      return { kind: "auth_required" };
    }
  }

  if (!active) {
    return { kind: "auth_required" };
  }

  if (active.status === "waiting_for_login") {
    const check = await bridge.getSession(active.sessionId, accountId);
    if (check.status !== "authenticated") {
      return { kind: "auth_pending" };
    }
  }

  return {
    kind: "ready",
    sessionId: active.sessionId,
    createdForCollect,
  };
}

async function evaluateUsageWithRetry(
  bridge: HostBrowserBridgeClient,
  sessionId: string,
  accountId: string,
  evalScript: string,
): Promise<EvaluateResult | undefined> {
  for (let attempt = 1; attempt <= EVALUATE_MAX_ATTEMPTS; attempt += 1) {
    try {
      const result = (await bridge.executeInSession(
        sessionId,
        accountId,
        "evaluate",
        evalScript,
      )) as EvaluateResult | undefined;
      if (result?.ok && result.data) {
        return result;
      }
    } catch (err) {
      if (err instanceof BridgeSessionNotFoundError) {
        return undefined;
      }
      const message = err instanceof Error ? err.message : String(err);
      if (attempt === EVALUATE_MAX_ATTEMPTS && /timeout/i.test(message)) {
        throw err;
      }
    }
    if (attempt < EVALUATE_MAX_ATTEMPTS) {
      await sleep(EVALUATE_RETRY_DELAY_MS);
    }
  }
  return undefined;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}
