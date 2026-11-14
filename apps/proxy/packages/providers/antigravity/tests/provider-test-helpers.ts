import type { AgentExecutionInput } from "@agent-runtime-gateway/core";
import type { ContentBlock } from "@agent-runtime-gateway/provider-antigravity/acp";
import type {
  AgentTurnResult,
  RunAgentTurnOptions,
  SessionConfigOption,
} from "@agent-runtime-gateway/provider-antigravity/acp";
import type { AntigravityAcpClient } from "../src/acp-client.js";

export class FakeAcpClient implements AntigravityAcpClient {
  public readonly calls: string[] = [];
  public lastBlocks?: readonly ContentBlock[];
  public firstPromptBlocks?: readonly ContentBlock[];
  public onPrompt: (options: RunAgentTurnOptions | undefined) => Promise<AgentTurnResult> = (
    options,
  ) => {
    options?.onEvent?.("agent_message_chunk", {
      content: { type: "text", text: "ACP response" },
    });
    return Promise.resolve({
      prompt: { stopReason: "end_turn" },
      agentText: "ACP response",
      thoughtText: "",
    });
  };

  public connect(): Promise<void> {
    this.calls.push("connect");
    return Promise.resolve();
  }

  public createSession(cwd: string): Promise<{ sessionId: string }> {
    this.calls.push(`new:${cwd}`);
    return Promise.resolve({ sessionId: "acp-session-1" });
  }

  public resumeSession(
    sessionId: string,
    cwd: string,
  ): Promise<Record<string, unknown>> {
    this.calls.push(`resume:${sessionId}:${cwd}`);
    return Promise.resolve({ sessionId });
  }

  public listSessions(): Promise<{ sessions: readonly { sessionId: string }[] }> {
    this.calls.push("list");
    return Promise.resolve({ sessions: [] });
  }

  public setSessionConfig(
    sessionId: string,
    configId: string,
    value: string | boolean,
  ): Promise<readonly SessionConfigOption[]> {
    this.calls.push(`config:${sessionId}:${configId}:${String(value)}`);
    return Promise.resolve([]);
  }

  public prompt(
    sessionId: string,
    blocks: readonly ContentBlock[],
    options?: RunAgentTurnOptions,
  ): Promise<AgentTurnResult> {
    this.calls.push(`prompt:${sessionId}`);
  if (this.firstPromptBlocks === undefined) {
    this.firstPromptBlocks = blocks;
  }
    this.lastBlocks = blocks;
    return this.onPrompt(options);
  }

  public cancelTurn(sessionId: string): void {
    this.calls.push(`cancel:${sessionId}`);
  }

  public closeSession(sessionId: string): Promise<void> {
    this.calls.push(`close:${sessionId}`);
    return Promise.resolve();
  }

  public terminate(): Promise<void> {
    this.calls.push("terminate");
    return Promise.resolve();
  }
}

export function executionInput(
  overrides: Partial<AgentExecutionInput> = {},
): AgentExecutionInput {
  return {
    credential: { type: "local", value: "" },
    model: "antigravity:gemini-3.8-flash",
    sessionId: "gateway-session-1",
    message: "Hello",
    workingDirectory: "/workspace",
    ...overrides,
  };
}
