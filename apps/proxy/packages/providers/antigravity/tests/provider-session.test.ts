import type { AgentEvent } from "@agent-runtime-gateway/core";
import type { AgentTurnResult } from "@agent-runtime-gateway/provider-antigravity/acp";
import { describe, expect, it } from "vitest";
import { AntigravityProvider } from "../src/provider.js";
import { executionInput, FakeAcpClient } from "./provider-test-helpers.js";

describe("AntigravityProvider session turns", () => {
  it("streams ACP message chunks and resumes the mapped session on the next run", async () => {
    const clients: FakeAcpClient[] = [];
    const provider = new AntigravityProvider({
      clientFactory: () => {
        const client = new FakeAcpClient();
        clients.push(client);
        return client;
      },
    });

    const firstRun = await provider.execute(executionInput());
    const firstEvents: AgentEvent[] = [];
    for await (const event of firstRun.events) firstEvents.push(event);
    const firstResult = await firstRun.wait();

    expect(firstResult).toEqual({ status: "completed", output: { text: "ACP response" } });
    expect(
      firstEvents.some(
        (event) => event.type === "message.delta" && event.delta === "ACP response",
      ),
    ).toBe(true);
    expect(clients[0]?.calls).toContain("config:acp-session-1:model:gemini-3.8-flash");
    expect(clients[0]?.calls).toContain("terminate");

    const secondRun = await provider.execute(executionInput({ message: "Follow up" }));
    await secondRun.wait();

    expect(clients[1]?.calls).toContain("resume:acp-session-1:/workspace");
    expect(clients[1]?.calls).toContain("prompt:acp-session-1");
  });

  it("keeps Gemini thought chunks out of the visible reply", async () => {
    const client = new FakeAcpClient();
    client.onPrompt = (options) => {
      options?.onEvent?.("agent_thought_chunk", {
        content: { type: "text", text: "Initial hypothesis: debate which tool to call." },
      });
      options?.onEvent?.("agent_message_chunk", {
        content: { type: "text", text: "Vou olhar o diff local." },
      });
      return Promise.resolve({
        prompt: { stopReason: "end_turn" },
        agentText: "Vou olhar o diff local.",
        thoughtText: "Initial hypothesis: debate which tool to call.",
      });
    };
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const run = await provider.execute(executionInput());
    const events: AgentEvent[] = [];
    for await (const event of run.events) events.push(event);

    const visible = events
      .filter((event) => event.type === "message.delta")
      .map((event) => event.delta)
      .join("");
    expect(visible).toBe("Vou olhar o diff local.");
    expect(visible).not.toContain("Initial hypothesis");
  });

  it("publishes ACP deltas before the prompt turn finishes", async () => {
    let emitDelta: (() => void) | undefined;
    let finishPrompt: (() => void) | undefined;
    const client = new FakeAcpClient();
    client.onPrompt = (options) =>
      new Promise<AgentTurnResult>((resolve) => {
        emitDelta = () =>
          options?.onEvent?.("agent_message_chunk", {
            content: { type: "text", text: "first delta" },
          });
        finishPrompt = () =>
          resolve({
            prompt: { stopReason: "end_turn" },
            agentText: "first delta",
            thoughtText: "",
          });
      });
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const run = await provider.execute(executionInput());
    const iterator = run.events[Symbol.asyncIterator]();

    await iterator.next();
    const deltaPromise = iterator.next();
    emitDelta?.();
    const delta = await deltaPromise;
    let resultSettled = false;
    const resultPromise = run.wait().then((result) => {
      resultSettled = true;
      return result;
    });

    expect(delta).toMatchObject({
      done: false,
      value: { type: "message.delta", delta: "first delta" },
    });
    expect(resultSettled).toBe(false);

    finishPrompt?.();
    await expect(resultPromise).resolves.toMatchObject({ status: "completed" });
  });

  it("maps cancellation to ACP session/cancel and a cancelled AgentResult", async () => {
    const client = hangingClient();
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const run = await provider.execute(executionInput());

    await run.cancel();
    const result = await run.wait();

    expect(client.calls).toContain("cancel:acp-session-1");
    expect(result.status).toBe("cancelled");
    expect(client.calls).toContain("terminate");
  });

  it("rejects an overlapping execution for the same gateway session", async () => {
    const client = hangingClient();
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const firstRun = await provider.execute(executionInput());

    await expect(provider.execute(executionInput())).rejects.toThrow(/turn in progress/i);
    await firstRun.cancel();
    await expect(firstRun.wait()).resolves.toMatchObject({ status: "cancelled" });
  });

  it("maps an aborted request signal to ACP cancellation", async () => {
    const client = hangingClient();
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const controller = new AbortController();
    const run = await provider.execute(executionInput({ signal: controller.signal }));

    controller.abort();

    await expect(run.wait()).resolves.toMatchObject({ status: "cancelled" });
    expect(client.calls).toContain("cancel:acp-session-1");
    expect(client.calls).toContain("terminate");
  });

  it("terminates the ACP process when the agent ignores cancellation", async () => {
    let finishPrompt: ((result: AgentTurnResult) => void) | undefined;
    const client = new FakeAcpClient();
    client.onPrompt = () =>
      new Promise<AgentTurnResult>((resolve) => {
        finishPrompt = resolve;
      });
    client.cancelTurn = (sessionId: string): void => {
      client.calls.push(`cancel:${sessionId}`);
    };
    client.terminate = (): Promise<void> => {
      client.calls.push("terminate");
      finishPrompt?.({
        prompt: { stopReason: "cancelled" },
        agentText: "",
        thoughtText: "",
      });
      return Promise.resolve();
    };
    const provider = new AntigravityProvider({
      clientFactory: () => client,
      cancelGraceMs: 5,
    });
    const controller = new AbortController();
    const run = await provider.execute(executionInput({ signal: controller.signal }));

    controller.abort();

    await expect(run.wait()).resolves.toMatchObject({ status: "cancelled" });
    expect(client.calls).toContain("cancel:acp-session-1");
    expect(client.calls).toContain("terminate");
  });
});

function hangingClient(): FakeAcpClient {
  let finishPrompt: ((result: AgentTurnResult) => void) | undefined;
  const client = new FakeAcpClient();
  client.onPrompt = () =>
    new Promise<AgentTurnResult>((resolve) => {
      finishPrompt = resolve;
    });
  client.cancelTurn = (sessionId: string): void => {
    client.calls.push(`cancel:${sessionId}`);
    finishPrompt?.({
      prompt: { stopReason: "cancelled" },
      agentText: "",
      thoughtText: "",
    });
  };
  return client;
}
