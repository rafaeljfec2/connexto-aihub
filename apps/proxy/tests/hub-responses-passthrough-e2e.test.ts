import { buildApp } from "@agent-runtime-gateway/api";
import { CursorProvider } from "../packages/providers/cursor/src/provider.js";
import request from "supertest";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  initializeGatewayServices,
  resetGatewayServicesForTest,
} from "../apps/api/src/services/gateway.js";

describe("Hub POST /v1/responses opaque passthrough", () => {
  afterEach(() => resetGatewayServicesForTest());

  it("forwards tools to edge-owned runner without hub directives", async () => {
    const edgeOwnedRunner = vi.fn(() =>
      Promise.resolve({
        agentId: "edge-agent",
        runId: "edge-run",
        stream: (async function* () {
          await Promise.resolve();
          yield {
            type: "tool_call" as const,
            toolCall: { name: "probe_tool", arguments: {} },
          };
          yield { type: "done" as const, finalText: "", toolCalls: [] };
        })(),
      }),
    );
    const cursorProvider = new CursorProvider({ edgeOwnedRunner });
    initializeGatewayServices(
      { HOST: "127.0.0.1", SESSION_STORAGE_DRIVER: "memory" },
      { cursorProvider },
    );

    const app = buildApp({
      HOST: "127.0.0.1",
      GATEWAY_API_TOKEN: "gateway-test-token",
      CURSOR_EXECUTION_MODE: "auto",
    });

    const payload = {
      model: "composer-2.5",
      instructions: "passthrough probe",
      tool_choice: "auto",
      tools: [{ type: "custom", name: "probe_tool" }],
      input: [{ type: "input_text", text: "use probe_tool" }],
      stream: false,
    };

    const res = await request(app)
      .post("/v1/responses")
      .set("X-Gateway-API-Key", "gateway-test-token")
      .set("Authorization", "Bearer cursor-test-key")
      .send(payload);

    expect(res.status).toBe(200);
    expect(edgeOwnedRunner).toHaveBeenCalledWith(
      expect.objectContaining({
        apiKey: "cursor-test-key",
        prompt: expect.not.stringContaining("SYSTEM_DIRECTIVE"),
        clientTools: [expect.objectContaining({ name: "probe_tool" })],
      }),
    );
  });
});
