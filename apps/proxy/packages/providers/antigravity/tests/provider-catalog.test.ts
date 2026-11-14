import { afterEach, describe, expect, it, vi } from "vitest";
import { AntigravityProvider } from "../src/provider.js";
import { executionInput, FakeAcpClient } from "./provider-test-helpers.js";

describe("AntigravityProvider catalog", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("lists the namespaced ACP models without starting an agent process", async () => {
    const clientFactory = vi.fn(() => new FakeAcpClient());
    const provider = new AntigravityProvider({ clientFactory });

    const models = await provider.listModels();

    expect(models.map((model) => model.id)).toEqual([
      "antigravity:gemini-3.8-flash",
      "antigravity:claude-sonnet-4-6",
      "antigravity:claude-opus-4-6-thinking",
      "antigravity:gpt-oss-120b-medium",
    ]);
    expect(clientFactory).not.toHaveBeenCalled();
  });

  it("rejects non-local credentials before spawning the ACP process", async () => {
    const clientFactory = vi.fn(() => new FakeAcpClient());
    const provider = new AntigravityProvider({ clientFactory });

    await expect(
      provider.execute(
        executionInput({ credential: { type: "api_key", value: "not-forwarded" } }),
      ),
    ).rejects.toThrow(/local/i);
    expect(clientFactory).not.toHaveBeenCalled();
  });

  it("allows reasoningEffort for Claude models without throwing and sets thought_level only for gemini-3.8-flash", async () => {
    const client = new FakeAcpClient();
    const provider = new AntigravityProvider({ clientFactory: () => client });

    const claudeRun = await provider.execute(
      executionInput({
        model: "antigravity:claude-sonnet-4-6",
        reasoningEffort: "medium",
      }),
    );
    const claudeResult = await claudeRun.wait();
    expect(claudeResult.status).toBe("completed");
    expect(client.calls.some((call) => call.includes("config:") && call.includes("thought_level"))).toBe(
      false,
    );

    const geminiRun = await provider.execute(
      executionInput({
        sessionId: "session-gemini",
        model: "antigravity:gemini-3.8-flash",
        reasoningEffort: "high",
      }),
    );
    const geminiResult = await geminiRun.wait();
    expect(geminiResult.status).toBe("completed");
    expect(
      client.calls.some((call) => call.includes("config:") && call.includes("thought_level:high")),
    ).toBe(true);
  });
});
