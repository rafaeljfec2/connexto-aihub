import {
  BrowserUnavailableError,
  ChallengeRequiredError,
  CollectionFailedError,
  SessionExpiredError,
} from "./playwright-browser.js";
import type { CursorUsageRuntime } from "./cursor-usage-runtime.js";
import {
  isCollectableSessionRecord,
  isUsableStorageState,
} from "./session-validation.js";
import type { CursorUsageCollectResult } from "./types.js";
import {
  DEFAULT_USAGE_ANALYTICS_SUFFIX,
  DEFAULT_USAGE_STATUS_SUFFIX,
  normalizeCursorUsagePayload,
  resolveCollectApiUrl,
} from "./usage-normalize.js";

export async function collectUsageLocally(
  runtime: CursorUsageRuntime,
  accountId: string,
): Promise<CursorUsageCollectResult> {
  const session = await runtime.sessionStore.resolve(accountId);
  if (runtime.browserPool.hasActiveAuthForAccount(accountId)) {
    if (
      session !== undefined &&
      session.status === "authenticated" &&
      isCollectableSessionRecord(
        session,
        runtime.browserPool.hasAuthProfile(accountId),
      )
    ) {
      await runtime.browserPool.closeAuthSessionsForAccount(accountId);
    } else {
      return {
        status: "authentication_pending",
        provider: "cursor",
        accountId,
      };
    }
  }

  const profilePresent = runtime.browserPool.hasAuthProfile(accountId);
  if (session === undefined || session.status !== "authenticated") {
    if (!profilePresent) {
      return {
        status: "authentication_required",
        provider: "cursor",
        accountId,
      };
    }
  } else if (!isCollectableSessionRecord(session, profilePresent)) {
    await runtime.sessionStore.delete(accountId);
    return {
      status: "authentication_required",
      provider: "cursor",
      accountId,
    };
  }

  try {
    const restoredState = session?.storageState ?? { cookies: [], origins: [] };
    const context = await runtime.browserPool.restoreAuthenticatedSession(
      accountId,
      restoredState,
    );
    await runtime.browserPool.primeCollectDashboard({
      context,
      dashboardUrl: runtime.collectDashboardUrl,
      urlPattern: runtime.usageUrlPattern,
      timeoutMs: runtime.collectTimeoutMs,
    });

    let usage: Record<string, unknown>;
    let collectionMethod: string;
    try {
      usage = await collectUsageViaApi(runtime, context);
      collectionMethod = "dashboard_api";
    } catch (apiError) {
      if (apiError instanceof SessionExpiredError) {
        throw apiError;
      }
      const raw = await runtime.browserPool.waitForUsageJson({
        context,
        dashboardUrl: runtime.collectDashboardUrl,
        urlPattern: runtime.usageUrlPattern,
        timeoutMs: runtime.collectTimeoutMs,
      });
      usage = normalizeUsagePayloadFromNetwork(raw);
      collectionMethod = "browser_network";
    }

    const storageState = await context.storageState();
    await context.close();
    const persistTarget =
      session ??
      ({
        accountId,
        status: "authenticated",
        touchedAt: runtime.now(),
      } as const);
    await runtime.sessionStore
      .setPersisted(
        {
          ...persistTarget,
          status: "authenticated",
          storageState,
        },
        {
          allowAuthProfileFallback:
            profilePresent && !isUsableStorageState(storageState),
        },
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
        collectionMethod,
      },
    };
  } catch (error) {
    return mapLocalCollectError(runtime, accountId, error);
  }
}

async function mapLocalCollectError(
  runtime: CursorUsageRuntime,
  accountId: string,
  error: unknown,
): Promise<CursorUsageCollectResult> {
  if (error instanceof ChallengeRequiredError) {
    return { status: "challenge_required", provider: "cursor", accountId };
  }
  if (error instanceof SessionExpiredError) {
    await runtime.sessionStore.delete(accountId);
    return { status: "session_expired", provider: "cursor", accountId };
  }
  if (error instanceof BrowserUnavailableError) {
    return { status: "browser_unavailable", provider: "cursor", accountId };
  }
  if (error instanceof CollectionFailedError) {
    return { status: "collection_failed", provider: "cursor", accountId };
  }
  const message = error instanceof Error ? error.message : String(error);
  if (/timeout/i.test(message)) {
    return { status: "timeout", provider: "cursor", accountId };
  }
  return { status: "collection_failed", provider: "cursor", accountId };
}

async function collectUsageViaApi(
  runtime: CursorUsageRuntime,
  context: import("playwright-core").BrowserContext,
): Promise<Record<string, unknown>> {
  const statusUrl = resolveCollectApiUrl(
    runtime.collectDashboardUrl,
    runtime.usageStatusUrl,
    DEFAULT_USAGE_STATUS_SUFFIX,
  );
  const referer = `${runtime.collectDashboardUrl.replace(/\/$/, "")}/`;
  const statusResponse = await context.request.get(statusUrl, {
    timeout: runtime.collectTimeoutMs,
    headers: {
      Accept: "application/json",
      Origin: "https://cursor.com",
      Referer: referer,
    },
  });
  if (!statusResponse.ok()) {
    if (statusResponse.status() === 401 || statusResponse.status() === 403) {
      throw new SessionExpiredError();
    }
    throw new CollectionFailedError(
      `usage-status responded ${statusResponse.status()}`,
    );
  }
  const usageStatus: unknown = await statusResponse.json();
  const analytics = await fetchOptionalAnalytics(runtime, context, referer);
  return normalizeCursorUsagePayload(usageStatus, analytics);
}

async function fetchOptionalAnalytics(
  runtime: CursorUsageRuntime,
  context: import("playwright-core").BrowserContext,
  referer: string,
): Promise<unknown> {
  const analyticsUrl = resolveCollectApiUrl(
    runtime.collectDashboardUrl,
    runtime.usageAnalyticsUrl,
    DEFAULT_USAGE_ANALYTICS_SUFFIX,
  );
  const timeout = Math.min(runtime.collectTimeoutMs, 30_000);
  const headers = {
    Accept: "application/json",
    Origin: "https://cursor.com",
    Referer: referer,
  } as const;

  if (analyticsUrl.includes("get-current-period-usage")) {
    const response = await context.request.post(analyticsUrl, {
      timeout,
      headers: { ...headers, "Content-Type": "application/json" },
      data: {},
    });
    return response.ok() ? response.json() : undefined;
  }

  const response = await context.request.get(analyticsUrl, {
    timeout,
    headers,
  });
  return response.ok() ? response.json() : undefined;
}

function normalizeUsagePayloadFromNetwork(raw: unknown): Record<string, unknown> {
  if (raw === null || typeof raw !== "object") {
    return { raw };
  }
  const body = raw as Record<string, unknown>;
  if (
    body.weeklyIncludedUsage !== undefined ||
    body.weekly_included_usage !== undefined ||
    body.monthlyUsage !== undefined ||
    body.weeklyIncludedUsageByProduct !== undefined ||
    body.availability !== undefined
  ) {
    return normalizeCursorUsagePayload(body);
  }
  return { ...body };
}
