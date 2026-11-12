import { describe, expect, it, vi } from "vitest";
import {
  DEFAULT_BRIDGE_USAGE_FETCH_TIMEOUT_MS,
  DEFAULT_BRIDGE_USAGE_OPTIONAL_FETCH_TIMEOUT_MS,
  buildHostBridgeUsageCollectScript,
  collectDashboardUsageFromFetch,
} from "../src/bridge-usage-collect.js";

/** HostBrowserBridgeClient defaultTimeoutMs — evaluate must finish under this. */
const BRIDGE_CLIENT_EVAL_CEILING_MS = 15_000;

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function hangingFetch(_input: string, init?: RequestInit): Promise<Response> {
  return new Promise<Response>((_resolve, reject) => {
    const signal = init?.signal;
    if (signal === undefined) {
      reject(new Error("missing abort signal"));
      return;
    }
    if (signal.aborted) {
      reject(new DOMException("Aborted", "AbortError"));
      return;
    }
    signal.addEventListener("abort", () => {
      reject(new DOMException("Aborted", "AbortError"));
    });
  });
}

describe("collectDashboardUsageFromFetch", () => {
  it("returns ok when usage-status succeeds while prepaid never resolves", async () => {
    const usageStatus = {
      availability: "available",
      weeklyIncludedUsage: { usedCents: 399, budgetCents: 16159 },
    };
    const fetchImpl = vi.fn(async (input: string, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("usage-status")) {
        return jsonResponse(usageStatus);
      }
      return hangingFetch(url, init);
    });

    const started = Date.now();
    const result = await collectDashboardUsageFromFetch(fetchImpl, 50, 40);
    const elapsedMs = Date.now() - started;

    expect(result.ok).toBe(true);
    if (!result.ok) {
      throw new Error("expected ok result");
    }
    expect(result.data.usageStatus).toEqual(usageStatus);
    expect(result.data.prepaid).toBeUndefined();
    // Wall-clock optional budget must cut enrichment off (not N endpoints × methods).
    expect(elapsedMs).toBeLessThan(BRIDGE_CLIENT_EVAL_CEILING_MS);
    expect(elapsedMs).toBeLessThan(250);
  });

  it("caps optional enrichment by a single wall-clock budget under the bridge ceiling", () => {
    const worstCaseWithUsageStatusMs =
      DEFAULT_BRIDGE_USAGE_FETCH_TIMEOUT_MS +
      DEFAULT_BRIDGE_USAGE_OPTIONAL_FETCH_TIMEOUT_MS;
    expect(worstCaseWithUsageStatusMs).toBeLessThan(BRIDGE_CLIENT_EVAL_CEILING_MS);
    // Explicitly refute the multi-endpoint × POST+GET mis-model (≥15s).
    const falseSequentialOptionalBudgetMs =
      3 * 2 * DEFAULT_BRIDGE_USAGE_OPTIONAL_FETCH_TIMEOUT_MS;
    expect(falseSequentialOptionalBudgetMs).toBeGreaterThanOrEqual(
      BRIDGE_CLIENT_EVAL_CEILING_MS,
    );
    expect(DEFAULT_BRIDGE_USAGE_OPTIONAL_FETCH_TIMEOUT_MS).toBeLessThan(
      falseSequentialOptionalBudgetMs,
    );
  });

  it("returns ok false when every endpoint fails or times out", async () => {
    const fetchImpl = vi.fn(async (input: string, init?: RequestInit) =>
      hangingFetch(input, init),
    );

    const result = await collectDashboardUsageFromFetch(fetchImpl, 20, 20);

    expect(result.ok).toBe(false);
    if (result.ok) {
      throw new Error("expected failure");
    }
    expect(result.error).toBe("no_matched_endpoint");
  });
});

describe("buildHostBridgeUsageCollectScript", () => {
  it("embeds AbortController, dual timeouts, parallel enrichment, and wall-clock race", () => {
    const script = buildHostBridgeUsageCollectScript(7_500, 2_000);
    expect(script).toContain("AbortController");
    expect(script).toContain("7500");
    expect(script).toContain("2000");
    expect(script).toContain("Promise.all");
    expect(script).toContain("raceWithTimeout");
    expect(script).toContain("get-user-prepaid-billing");
    expect(script).toContain("usage-status");
  });

  it("defaults timeouts to package constants", () => {
    const script = buildHostBridgeUsageCollectScript();
    expect(script).toContain(String(DEFAULT_BRIDGE_USAGE_FETCH_TIMEOUT_MS));
    expect(script).toContain(
      String(DEFAULT_BRIDGE_USAGE_OPTIONAL_FETCH_TIMEOUT_MS),
    );
  });
});
