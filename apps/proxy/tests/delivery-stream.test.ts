import { describe, expect, it } from "vitest";
import type { AgentEvent, AgentResult, AgentRun } from "@agent-runtime-gateway/core";
import { TextDecoder } from "node:util";
import { streamResponseRun, type ResponseDeliveryContext } from "../apps/api/src/routes/responses/delivery.js";

function stubRun(events: AgentEvent[], result: AgentResult): AgentRun {
  async function* gen() {
    await Promise.resolve();
    for (const event of events) {
      yield event;
    }
  }
  return {
    id: "run_test",
    events: gen(),
    wait: () => Promise.resolve(result),
    cancel: () => Promise.resolve(),
  };
}

function createCaptureResponse(): { res: { writableEnded: boolean }; getBody: () => string } {
  let body = "";
  const res = {
    headersSent: true,
    writableEnded: false,
    setHeader: () => undefined,
    write: (chunk: Uint8Array | string) => {
      body += typeof chunk === "string" ? chunk : new TextDecoder().decode(chunk);
      return true;
    },
    end: () => {
      res.writableEnded = true;
    },
  };
  return { res, getBody: () => body };
}

const baseContext = (signal: AbortSignal): ResponseDeliveryContext => ({
  id: "resp_delivery_test",
  created: 1_700_000_000,
  requestId: "req-delivery",
  principal: "test",
  sessionAffinity: "sess",
  target: { providerId: "cursor", modelId: "composer-2.5", credentialType: "api_key" },
  prepared: {
    model: "composer-2.5",
    promptChars: 12,
    stream: true,
    storeResponse: false,
  } as ResponseDeliveryContext["prepared"],
  providerStart: performance.now(),
  services: { responseStore: { set: () => undefined } } as unknown as ResponseDeliveryContext["services"],
  signal,
});

describe("streamResponseRun terminal SSE", () => {
  it("emits response.completed before end when the run fails after a text delta", async () => {
    const run = stubRun(
      [
        { type: "message.delta", runId: "r", delta: "partial" },
        { type: "run.failed", runId: "r", message: "boom" },
      ],
      { status: "failed", error: { message: "boom" } },
    );
    const { res, getBody } = createCaptureResponse();
    const abort = new AbortController();
    await streamResponseRun(run, res as never, baseContext(abort.signal));

    const body = getBody();
    expect(body).toContain("partial");
    expect(body).toContain("event: response.completed");
    expect(body).toContain('"status":"incomplete"');
    expect(body).toContain("[DONE]");
    expect(body.lastIndexOf("response.completed")).toBeGreaterThan(body.indexOf("partial"));
    expect(res.writableEnded).toBe(true);
  });
});
