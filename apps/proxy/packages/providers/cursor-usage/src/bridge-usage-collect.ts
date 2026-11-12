/** Per-request abort budget for primary usage-status fetches. */
export const DEFAULT_BRIDGE_USAGE_FETCH_TIMEOUT_MS = 8_000;

/**
 * Wall-clock budget for the entire optional prepaid/analytics enrichment phase
 * after a successful usage-status. Must keep
 * primary + optional under HostBrowserBridgeClient.defaultTimeoutMs (15s).
 */
export const DEFAULT_BRIDGE_USAGE_OPTIONAL_FETCH_TIMEOUT_MS = 2_500;

const USAGE_STATUS_ENDPOINTS = [
  "/dashboard/api/dashboard/v2/usage-status?scope=user",
  "/api/usage-status?scope=user",
  "/dashboard/api/usage-status?scope=user",
  "https://grok.com/dashboard/api/dashboard/v2/usage-status?scope=user",
  "/api/usage-summary",
  "https://cursor.com/api/usage-summary",
  "/api/dashboard/get-current-period-usage",
] as const;

const PREPAID_ENDPOINTS = [
  "/dashboard/api/dashboard/get-user-prepaid-billing",
  "/api/dashboard/get-user-prepaid-billing",
  "https://grok.com/dashboard/api/dashboard/get-user-prepaid-billing",
] as const;

const ANALYTICS_ENDPOINTS = [
  "/dashboard/api/dashboard/get-user-analytics",
  "/api/dashboard/get-user-analytics",
  "/api/dashboard/get-current-period-usage",
] as const;

export type BridgeUsageCollectFetch = (
  input: string,
  init?: RequestInit,
) => Promise<Response>;

export type BridgeUsageCollectData = {
  readonly usageStatus: Record<string, unknown>;
  readonly prepaid?: Record<string, unknown>;
  readonly analytics?: Record<string, unknown>;
};

export type BridgeUsageCollectResult =
  | { readonly ok: true; readonly data: BridgeUsageCollectData }
  | { readonly ok: false; readonly error: string };

function isUsefulRecord(
  value: Record<string, unknown> | null,
  requireNoErrorKey: boolean,
): value is Record<string, unknown> {
  if (value === null) {
    return false;
  }
  if (Object.keys(value).length === 0) {
    return false;
  }
  if (requireNoErrorKey && value.error !== undefined) {
    return false;
  }
  return true;
}

export async function fetchJsonWithTimeout(
  fetchImpl: BridgeUsageCollectFetch,
  url: string,
  method: string,
  timeoutMs: number,
): Promise<Record<string, unknown> | null> {
  const controller = new AbortController();
  const timer = setTimeout(() => {
    controller.abort();
  }, timeoutMs);
  try {
    const response = await fetchImpl(url, {
      method,
      credentials: "include",
      headers: { Accept: "application/json" },
      signal: controller.signal,
    });
    if (!response.ok) {
      return null;
    }
    const data: unknown = await response.json();
    if (data !== null && typeof data === "object") {
      return data as Record<string, unknown>;
    }
    return null;
  } catch {
    return null;
  } finally {
    clearTimeout(timer);
  }
}

async function fetchFirstUseful(
  fetchImpl: BridgeUsageCollectFetch,
  endpoints: readonly string[],
  timeoutMs: number,
  options: {
    readonly requireNoErrorKey: boolean;
    readonly preferPostThenGet: boolean;
    readonly postWhenPathIncludes?: string;
  },
): Promise<Record<string, unknown> | null> {
  for (const endpoint of endpoints) {
    if (options.preferPostThenGet) {
      let value = await fetchJsonWithTimeout(
        fetchImpl,
        endpoint,
        "POST",
        timeoutMs,
      );
      if (!isUsefulRecord(value, options.requireNoErrorKey)) {
        value = await fetchJsonWithTimeout(fetchImpl, endpoint, "GET", timeoutMs);
      }
      if (isUsefulRecord(value, options.requireNoErrorKey)) {
        return value;
      }
      continue;
    }

    const method =
      options.postWhenPathIncludes !== undefined &&
      endpoint.includes(options.postWhenPathIncludes)
        ? "POST"
        : "GET";
    const value = await fetchJsonWithTimeout(
      fetchImpl,
      endpoint,
      method,
      timeoutMs,
    );
    if (isUsefulRecord(value, options.requireNoErrorKey)) {
      return value;
    }
  }
  return null;
}

export async function raceWithTimeout<T>(
  promise: Promise<T>,
  timeoutMs: number,
  fallback: T,
): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      promise,
      new Promise<T>((resolve) => {
        timer = setTimeout(() => {
          resolve(fallback);
        }, timeoutMs);
      }),
    ]);
  } finally {
    if (timer !== undefined) {
      clearTimeout(timer);
    }
  }
}

export async function collectDashboardUsageFromFetch(
  fetchImpl: BridgeUsageCollectFetch,
  timeoutMs: number = DEFAULT_BRIDGE_USAGE_FETCH_TIMEOUT_MS,
  optionalTimeoutMs: number = DEFAULT_BRIDGE_USAGE_OPTIONAL_FETCH_TIMEOUT_MS,
): Promise<BridgeUsageCollectResult> {
  const usageStatus = await fetchFirstUseful(
    fetchImpl,
    USAGE_STATUS_ENDPOINTS,
    timeoutMs,
    {
      requireNoErrorKey: false,
      preferPostThenGet: false,
      postWhenPathIncludes: "get-current-period-usage",
    },
  );

  if (usageStatus !== null) {
    const enrichment = await raceWithTimeout(
      Promise.all([
        fetchFirstUseful(fetchImpl, PREPAID_ENDPOINTS, optionalTimeoutMs, {
          requireNoErrorKey: true,
          preferPostThenGet: true,
        }),
        fetchFirstUseful(fetchImpl, ANALYTICS_ENDPOINTS, optionalTimeoutMs, {
          requireNoErrorKey: false,
          preferPostThenGet: true,
        }),
      ]).then(([prepaid, analytics]) => ({ prepaid, analytics })),
      optionalTimeoutMs,
      { prepaid: null, analytics: null },
    );
    return {
      ok: true,
      data: {
        usageStatus,
        prepaid: enrichment.prepaid ?? undefined,
        analytics: enrichment.analytics ?? undefined,
      },
    };
  }

  const prepaid = await fetchFirstUseful(
    fetchImpl,
    PREPAID_ENDPOINTS,
    timeoutMs,
    {
      requireNoErrorKey: true,
      preferPostThenGet: true,
    },
  );
  const analytics = await fetchFirstUseful(
    fetchImpl,
    ANALYTICS_ENDPOINTS,
    timeoutMs,
    {
      requireNoErrorKey: false,
      preferPostThenGet: true,
    },
  );

  if (prepaid !== null || analytics !== null) {
    return {
      ok: true,
      data: {
        usageStatus: {},
        prepaid: prepaid ?? undefined,
        analytics: analytics ?? undefined,
      },
    };
  }

  return { ok: false, error: "no_matched_endpoint" };
}

/**
 * Browser-side collect script for CDP evaluate.
 * Mirrors {@link collectDashboardUsageFromFetch}: primary usage-status, then
 * optional enrichment hard-capped by a single wall-clock budget.
 */
export function buildHostBridgeUsageCollectScript(
  timeoutMs: number = DEFAULT_BRIDGE_USAGE_FETCH_TIMEOUT_MS,
  optionalTimeoutMs: number = DEFAULT_BRIDGE_USAGE_OPTIONAL_FETCH_TIMEOUT_MS,
): string {
  const statusEndpointsJson = JSON.stringify(USAGE_STATUS_ENDPOINTS);
  const prepaidEndpointsJson = JSON.stringify(PREPAID_ENDPOINTS);
  const analyticsEndpointsJson = JSON.stringify(ANALYTICS_ENDPOINTS);

  return `
        (async () => {
          const timeoutMs = ${timeoutMs};
          const optionalTimeoutMs = ${optionalTimeoutMs};
          const statusEndpoints = ${statusEndpointsJson};
          const prepaidEndpoints = ${prepaidEndpointsJson};
          const analyticsEndpoints = ${analyticsEndpointsJson};
          try {
            const fetchJson = async (url, method, limitMs) => {
              const controller = new AbortController();
              const timer = setTimeout(() => controller.abort(), limitMs);
              try {
                const res = await fetch(url, {
                  method,
                  credentials: "include",
                  headers: { "Accept": "application/json" },
                  signal: controller.signal
                });
                if (res.ok) {
                  const data = await res.json();
                  if (data && typeof data === "object") return data;
                }
              } catch {}
              finally {
                clearTimeout(timer);
              }
              return null;
            };

            const isUseful = (value, requireNoErrorKey) => {
              if (!value || typeof value !== "object") return false;
              if (Object.keys(value).length === 0) return false;
              if (requireNoErrorKey && value.error !== undefined) return false;
              return true;
            };

            const raceWithTimeout = (promise, limitMs, fallback) => new Promise((resolve) => {
              const timer = setTimeout(() => resolve(fallback), limitMs);
              promise.then(
                (value) => { clearTimeout(timer); resolve(value); },
                () => { clearTimeout(timer); resolve(fallback); }
              );
            });

            const fetchFirstUseful = async (endpoints, limitMs, preferPostThenGet, requireNoErrorKey, postWhenPathIncludes) => {
              for (const ep of endpoints) {
                if (preferPostThenGet) {
                  let value = await fetchJson(ep, "POST", limitMs);
                  if (!isUseful(value, requireNoErrorKey)) {
                    value = await fetchJson(ep, "GET", limitMs);
                  }
                  if (isUseful(value, requireNoErrorKey)) return value;
                  continue;
                }
                const method = postWhenPathIncludes && ep.includes(postWhenPathIncludes) ? "POST" : "GET";
                const value = await fetchJson(ep, method, limitMs);
                if (isUseful(value, requireNoErrorKey)) return value;
              }
              return null;
            };

            const usageStatus = await fetchFirstUseful(
              statusEndpoints,
              timeoutMs,
              false,
              false,
              "get-current-period-usage"
            );

            if (usageStatus) {
              const enrichment = await raceWithTimeout(
                Promise.all([
                  fetchFirstUseful(prepaidEndpoints, optionalTimeoutMs, true, true, null),
                  fetchFirstUseful(analyticsEndpoints, optionalTimeoutMs, true, false, null)
                ]).then(([prepaid, analytics]) => ({ prepaid, analytics })),
                optionalTimeoutMs,
                { prepaid: null, analytics: null }
              );
              return {
                ok: true,
                data: {
                  usageStatus,
                  prepaid: enrichment.prepaid ?? undefined,
                  analytics: enrichment.analytics ?? undefined
                }
              };
            }

            const prepaid = await fetchFirstUseful(prepaidEndpoints, timeoutMs, true, true, null);
            const analytics = await fetchFirstUseful(analyticsEndpoints, timeoutMs, true, false, null);
            if (prepaid || analytics) {
              return {
                ok: true,
                data: {
                  usageStatus: {},
                  prepaid: prepaid ?? undefined,
                  analytics: analytics ?? undefined
                }
              };
            }
          } catch (e) {
            return { ok: false, error: e && e.message ? e.message : String(e) };
          }
          return { ok: false, error: "no_matched_endpoint" };
        })()
      `;
}
