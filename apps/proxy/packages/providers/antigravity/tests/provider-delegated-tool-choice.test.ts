import type { AgentTurnResult } from "@agent-runtime-gateway/provider-antigravity/acp";
import type { AntigravityMcpBridgeOptions } from "../src/mcp/types.js";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AntigravityProvider } from "../src/provider.js";
import { executionInput, FakeAcpClient } from "./provider-test-helpers.js";

const createdBridges = vi.hoisted(() => [] as AntigravityMcpBridgeOptions[]);

vi.mock("../src/mcp/index.js", () => {
  class AntigravityMcpBridge {
    public constructor(options: AntigravityMcpBridgeOptions) {
      createdBridges.push(options);
    }

    public start(): Promise<{ name: string; command: string; args: readonly string[] }> {
      return Promise.resolve({ name: "connexto_tools", command: "node", args: [] });
    }

    public dispose(): Promise<void> {
      return Promise.resolve();
    }
  }

  return { AntigravityMcpBridge };
});

describe("AntigravityProvider delegated tool_choice", () => {
  afterEach(() => {
    createdBridges.length = 0;
  });

  it("does not complete when tool_choice names a tool that was not recorded", async () => {
    const client = hangingPrompt();
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const run = await provider.execute(
      executionInput({
        tools: [
          { name: "required_tool", parameters: {} },
          { name: "other_tool", parameters: {} },
        ],
        toolChoice: { type: "function", function: { name: "required_tool" } },
      }),
    );

    recordTool("other_tool");
    const result = await run.wait();

    expect(result.status).toBe("failed");
  });

  it("does not complete when the prompt fails after a different tool was recorded", async () => {
    let rejectPrompt: ((error: Error) => void) | undefined;
    const client = new FakeAcpClient();
    client.onPrompt = () =>
      new Promise<AgentTurnResult>((_resolve, reject) => {
        rejectPrompt = reject;
      });
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const run = await provider.execute(
      executionInput({
        sessionId: "gateway-session-catch",
        tools: [
          { name: "required_tool", parameters: {} },
          { name: "other_tool", parameters: {} },
        ],
        toolChoice: { type: "function", function: { name: "required_tool" } },
      }),
    );

    recordTool("other_tool");
    rejectPrompt?.(new Error("upstream prompt failed"));
    const result = await run.wait();

    expect(result.status).toBe("failed");
  });

  it("completes a delegated client tool that tool_choice requested", async () => {
    const client = hangingPrompt();
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const run = await provider.execute(
      executionInput({
        sessionId: "gateway-session-match",
        tools: [{ name: "required_tool", parameters: {} }],
        toolChoice: { type: "function", function: { name: "required_tool" } },
      }),
    );

    recordTool("required_tool");
    const result = await run.wait();

    expect(result.status).toBe("completed");
    expect(result.output?.toolCalls?.[0]?.name).toBe("required_tool");
  });

  it("completes a delegated client tool when tool_choice is unset", async () => {
    const client = hangingPrompt();
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const run = await provider.execute(
      executionInput({
        sessionId: "gateway-session-auto",
        tools: [{ name: "other_tool", parameters: {} }],
      }),
    );

    recordTool("other_tool");
    const result = await run.wait();

    expect(result.status).toBe("completed");
    expect(result.output?.toolCalls?.[0]?.name).toBe("other_tool");
  });

  it("calls exec after a status sentence when the first turn stops without a tool", async () => {
    let prompts = 0;
    const client = new FakeAcpClient();
    client.onPrompt = () => {
      prompts += 1;
      if (prompts === 1) {
        return Promise.resolve({
          prompt: { stopReason: "end_turn" },
          agentText: "Vou verificar o status das alterações locais.",
          thoughtText: "",
        });
      }
      recordTool("exec");
      return new Promise<AgentTurnResult>(() => {});
    };
    const provider = new AntigravityProvider({ clientFactory: () => client });
    const run = await provider.execute(
      executionInput({
        sessionId: "gateway-session-follow-up",
        tools: [{ name: "exec", callKind: "custom", inputFormat: "grammar" }],
      }),
    );

    const result = await run.wait();

    expect(prompts).toBe(2);
    expect(result.status).toBe("completed");
    expect(result.output?.toolCalls?.[0]?.name).toBe("exec");
  });
});

function recordTool(name: string): void {
  const bridge = createdBridges.at(-1);
  bridge?.onToolCall?.({ callId: `call_${name}`, name, arguments: {} });
}

function hangingPrompt(): FakeAcpClient {
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
  client.terminate = (): Promise<void> => {
    client.calls.push("terminate");
    finishPrompt?.({
      prompt: { stopReason: "cancelled" },
      agentText: "",
      thoughtText: "",
    });
    return Promise.resolve();
  };
  return client;
}
