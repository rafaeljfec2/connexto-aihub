import { AccountLockMap } from "./account-lock.js";
import { HostBrowserBridgeClient } from "./host-bridge-client.js";
import {
  PlaywrightBrowserPool,
  type PlaywrightBrowserPool as BrowserPoolType,
} from "./playwright-browser.js";
import { CursorUsageSessionStore } from "./session-store.js";
import type {
  CursorUsageAuthOperation,
  CursorUsageServiceOptions,
} from "./types.js";
import { DEFAULT_COLLECT_DASHBOARD_URL } from "./usage-normalize.js";

const DEFAULT_LOGIN = "https://cursor.com/dashboard";
const DEFAULT_USAGE_PATTERN =
  /usage-summary|get-current-period|get-filtered-usage|usage-status|get-user-analytics/i;
const DEFAULT_AUTHENTICATED_URL_PATTERN =
  /(^https?:\/\/)?(?:([^/]+\.)?cursor\.com\/(dashboard|settings|agents)|grok\.com\/dashboard)(\/?|$|\?)/i;

export type CursorUsageRuntime = {
  readonly sessionStore: CursorUsageSessionStore;
  readonly browserPool: BrowserPoolType;
  readonly locks: AccountLockMap;
  readonly authOperations: Map<string, CursorUsageAuthOperation>;
  readonly liveContexts: Map<string, import("playwright-core").BrowserContext>;
  readonly authOperationTtlMs: number;
  readonly collectTimeoutMs: number;
  readonly loginUrl: string;
  readonly collectDashboardUrl: string;
  readonly usageStatusUrl?: string;
  readonly usageAnalyticsUrl?: string;
  readonly usageUrlPattern: RegExp;
  readonly collectReadyUrlPattern: RegExp;
  readonly interactionUrlBase: string;
  readonly authHeadless: boolean;
  readonly hostBridgeClient?: HostBrowserBridgeClient;
  readonly now: () => number;
};

export function createCursorUsageRuntime(
  options: CursorUsageServiceOptions = {},
): CursorUsageRuntime {
  const now = options.now ?? Date.now;
  const browserPool =
    options.browserPool ??
    new PlaywrightBrowserPool({
      chromiumExecutablePath: options.chromiumExecutablePath,
      cdpEndpoint: options.cdpEndpoint,
      cdpFallback: options.cdpFallback,
      chromiumNoSandbox: options.chromiumNoSandbox,
      collectHeadless: options.collectHeadless,
      authHeadless: options.authHeadless,
      authUserDataDirRoot: options.authUserDataDirRoot,
      authViewportWidth: options.authViewportWidth,
      authViewportHeight: options.authViewportHeight,
    });

  let hostBridgeClient: HostBrowserBridgeClient | undefined;
  if (options.hostBridgeClient) {
    hostBridgeClient = options.hostBridgeClient;
  } else if (options.hostBridgeUrl ?? process.env.BROWSER_BRIDGE_URL) {
    hostBridgeClient = new HostBrowserBridgeClient({
      baseUrl: options.hostBridgeUrl,
      token: options.hostBridgeToken,
    });
  }

  const authenticatedUrlPattern =
    options.authenticatedUrlPattern ?? DEFAULT_AUTHENTICATED_URL_PATTERN;

  return {
    now,
    sessionStore: new CursorUsageSessionStore({
      maxSessions: options.maxSessions,
      ttlMs: options.sessionTtlMs,
      store: options.store,
      now,
    }),
    browserPool,
    locks: new AccountLockMap(),
    authOperations: new Map<string, CursorUsageAuthOperation>(),
    liveContexts: new Map(),
    authOperationTtlMs: options.authOperationTtlMs ?? 30 * 60_000,
    collectTimeoutMs: options.collectTimeoutMs ?? 60_000,
    loginUrl: options.loginUrl ?? DEFAULT_LOGIN,
    collectDashboardUrl: options.dashboardUrl ?? DEFAULT_COLLECT_DASHBOARD_URL,
    usageStatusUrl: options.usageStatusUrl,
    usageAnalyticsUrl: options.usageAnalyticsUrl,
    usageUrlPattern: options.usageResponseUrlPattern ?? DEFAULT_USAGE_PATTERN,
    collectReadyUrlPattern:
      options.collectReadyUrlPattern ?? authenticatedUrlPattern,
    interactionUrlBase: options.interactionUrlBase ?? "",
    authHeadless: options.authHeadless !== false,
    hostBridgeClient,
  };
}
