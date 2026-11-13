import type { AgentOptions, RunResult, RunStatus, SDKMessage } from "@cursor/sdk";
import { describe, expect, it, vi } from "vitest";
import { CLIENT_MCP_SERVER_NAME } from "../src/client-mcp/constants.js";
import {
  createCursorSdkInProcessCompletion,
  type CursorSdkInProcessDependencies,
  type CursorSdkSendOptions,
} from "../src/sdk-inprocess-runner.js";
import type { CursorTextEvent } from "../src/engine/types.js";
describe("createCursorSdkInProcessCompletion client tools", () => {
  it("forwards only the first client tool from an SDK assistant tool batch", async () => {
    let runStatus: RunStatus = "running";
    const run = {
      id: "run-4",
      get status() {
        return runStatus;
      },
      async *stream(): AsyncGenerator<SDKMessage> {
        await Promise.resolve();
        yield {
          type: "assistant",
          agent_id: "agent-4",
          run_id: "run-4",
          message: {
            role: "assistant",
            content: [
              {
                type: "tool_use",
                id: "call-read",
                name: "mcp",
                input: {
                  providerIdentifier: "client",
                  toolName: "read",
                  args: { path: "a.txt" },
                },
              },
              {
                type: "tool_use",
                id: "call-write",
                name: "mcp",
                input: {
                  providerIdentifier: "client",
                  toolName: "write",
                  args: { path: "b.txt", fileText: "hello" },
                },
              },
            ],
          },
        };
      },
      wait: vi.fn(() => Promise.resolve({ id: "run-4", status: "cancelled" as const })),
      cancel: vi.fn(() => {
        runStatus = "cancelled";
        return Promise.resolve();
      }),
    };
    const agent = {
      agentId: "agent-4",
      send: vi.fn(() => Promise.resolve(run)),
      close: vi.fn(),
    };
    const dependencies: CursorSdkInProcessDependencies = {
      createAgent: vi.fn(() => Promise.resolve(agent)),
      deleteAgent: vi.fn(() => Promise.resolve()),
    };

    const completion = await createCursorSdkInProcessCompletion(
      {
        apiKey: "cursor-key",
        model: "composer-2.5",
        agentName: "gateway-test",
        prompt: "Read and write files",
        clientTools: [{ name: "read" }, { name: "write" }],
      },
      dependencies,
    );
    const events: CursorTextEvent[] = [];
    for await (const event of completion.stream) events.push(event);

    expect(events.filter((event) => event.type === "tool_call")).toEqual([
      {
        type: "tool_call",
        toolCall: {
          name: "read",
          arguments: { path: "a.txt" },
          sdkCallId: "call-read",
        },
      },
    ]);
    expect(run.cancel).toHaveBeenCalled();
  });

  it("rejects MCP calls that do not target the client provider", async () => {
    const run = {
      id: "run-6",
      status: "running" as const,
      async *stream(): AsyncGenerator<SDKMessage> {
        yield {
          type: "tool_call",
          agent_id: "agent-6",
          run_id: "run-6",
          call_id: "call-6",
          name: "mcp",
          status: "running",
          args: {
            providerIdentifier: "github",
            toolName: "create_issue",
            args: {},
          },
        };
      },
      wait: vi.fn(),
      cancel: vi.fn(() => Promise.resolve()),
    };
    const dependencies: CursorSdkInProcessDependencies = {
      createAgent: vi.fn(() =>
        Promise.resolve({
          agentId: "agent-6",
          send: () => Promise.resolve(run),
          close: vi.fn(),
        }),
      ),
      deleteAgent: vi.fn(() => Promise.resolve()),
    };

    const completion = await createCursorSdkInProcessCompletion(
      {
        apiKey: "cursor-key",
        model: "composer-2.5",
        agentName: "gateway-test",
        prompt: "Open issue",
        clientTools: [{ name: "mcp__github__create_issue" }],
      },
      dependencies,
    );

    await expect(async () => {
      for await (const _event of completion.stream) {
        // drain
      }
    }).rejects.toMatchObject({ code: "cursor_invalid_client_tool" });
  });

  it("forwards a client tool even when SDK cancel fails during cleanup", async () => {
    const toolCallMessage: SDKMessage = {
      type: "tool_call",
      agent_id: "agent-8",
      run_id: "run-8",
      call_id: "call-8",
      name: "mcp",
      status: "running",
      args: {
        providerIdentifier: "client",
        toolName: "write",
        args: { path: "note.txt" },
      },
    };
    const run = {
      id: "run-8",
      status: "running" as const,
      async *stream(): AsyncGenerator<SDKMessage> {
        yield toolCallMessage;
      },
      wait: vi.fn(() => Promise.resolve({ id: "run-8", status: "running" as const })),
      cancel: vi.fn(() => Promise.reject(new Error("cancel failed"))),
    };
    const dependencies: CursorSdkInProcessDependencies = {
      createAgent: vi.fn(() =>
        Promise.resolve({
          agentId: "agent-8",
          send: () => Promise.resolve(run),
          close: vi.fn(),
        }),
      ),
      deleteAgent: vi.fn(() => Promise.resolve()),
    };

    const completion = await createCursorSdkInProcessCompletion(
      {
        apiKey: "cursor-key",
        model: "composer-2.5",
        agentName: "gateway-test",
        prompt: "Write note",
        clientTools: [{ name: "write" }],
      },
      dependencies,
    );

    const events: CursorTextEvent[] = [];
    for await (const event of completion.stream) {
      events.push(event);
    }
    expect(events.filter((event) => event.type === "tool_call")).toEqual([
      {
        type: "tool_call",
        toolCall: {
          name: "write",
          arguments: { path: "note.txt", fileText: undefined },
          sdkCallId: "call-8",
        },
      },
    ]);
    expect(run.cancel).toHaveBeenCalled();
  });

  it("forwards invalid exec payloads as a contract correction at the runner boundary", async () => {
    const forbidden =
      'await tools.client_write({path:"secret.env",fileText:"leak"}); text("ok");';
    let runStatus: RunStatus = "running";
    const toolCallMessage: SDKMessage = {
      type: "tool_call",
      agent_id: "agent-9",
      run_id: "run-9",
      call_id: "call-9",
      name: "mcp",
      status: "running",
      args: {
        providerIdentifier: "client",
        toolName: "exec",
        args: { input: forbidden },
      },
    };
    const run = {
      id: "run-9",
      get status() {
        return runStatus;
      },
      async *stream(): AsyncGenerator<SDKMessage> {
        yield toolCallMessage;
      },
      wait: vi.fn(() => Promise.resolve({ id: "run-9", status: "cancelled" as const })),
      cancel: vi.fn(() => {
        runStatus = "cancelled";
        return Promise.resolve();
      }),
    };
    const dependencies: CursorSdkInProcessDependencies = {
      createAgent: vi.fn(() =>
        Promise.resolve({
          agentId: "agent-9",
          send: () => Promise.resolve(run),
          close: vi.fn(),
        }),
      ),
      deleteAgent: vi.fn(() => Promise.resolve()),
    };

    const completion = await createCursorSdkInProcessCompletion(
      {
        apiKey: "cursor-key",
        model: "composer-2.5",
        agentName: "gateway-test",
        prompt: "Leak secrets",
        clientTools: [{ name: "exec", inputFormat: "grammar" }],
      },
      dependencies,
    );
    const events: CursorTextEvent[] = [];
    for await (const event of completion.stream) events.push(event);

    const toolEvent = events.find((event) => event.type === "tool_call");
    expect(toolEvent?.type).toBe("tool_call");
    if (toolEvent?.type !== "tool_call") throw new Error("expected tool_call event");
    expect(toolEvent.toolCall.name).toBe("exec");
    expect(toolEvent.toolCall.rawInput).toMatch(/^text\(/);
    expect(toolEvent.toolCall.rawInput).toContain("CONTRACT VIOLATION");
    expect(toolEvent.toolCall.rawInput).not.toContain("await tools.client_write");
    expect(events.at(-1)).toMatchObject({
      type: "done",
      toolCalls: [expect.objectContaining({ rawInput: expect.stringContaining("CONTRACT VIOLATION") })],
    });
  });

  it("returns the client tool as soon as the MCP callback arrives, without waiting for the SDK stream to end", async () => {
    let sendOptions: CursorSdkSendOptions | undefined;
    const run = {
      id: "run-7",
      status: "running" as const,
      async *stream(): AsyncGenerator<SDKMessage> {
        await new Promise<void>(() => undefined);
      },
      wait: vi.fn(),
      cancel: vi.fn(() => Promise.resolve()),
    };
    const dependencies: CursorSdkInProcessDependencies = {
      createAgent: vi.fn(() =>
        Promise.resolve({
          agentId: "agent-7",
          send: (_message: unknown, options?: CursorSdkSendOptions) => {
            sendOptions = options;
            return Promise.resolve(run);
          },
          close: vi.fn(),
        }),
      ),
      deleteAgent: vi.fn(() => Promise.resolve()),
    };

    const completion = await createCursorSdkInProcessCompletion(
      {
        apiKey: "cursor-key",
        model: "composer-2.5",
        agentName: "gateway-test",
        prompt: "Review the diff",
        clientTools: [{ name: "exec", inputFormat: "grammar" }],
      },
      dependencies,
    );

    const eventsPromise = (async () => {
      const events: CursorTextEvent[] = [];
      for await (const event of completion.stream) events.push(event);
      return events;
    })();

    const server = sendOptions?.mcpServers?.[CLIENT_MCP_SERVER_NAME] as
      | { env?: Record<string, string> }
      | undefined;
    const callbackUrl = server?.env?.CURSOR_SDK_BRIDGE_CALLBACK_URL;
    const token = server?.env?.CURSOR_SDK_BRIDGE_CALLBACK_TOKEN;
    expect(callbackUrl).toEqual(expect.stringContaining("http://127.0.0.1"));

    const response = await fetch(callbackUrl ?? "", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token ?? ""}`,
      },
      body: JSON.stringify({
        toolName: "exec",
        arguments: {
          input:
            'await tools.' + 'client_write({path:"x",fileText:"y"}); text("ok");',
        },
      }),
    });
    expect(response.ok).toBe(true);

    const events = await eventsPromise;
    const toolEvent = events.find((event) => event.type === "tool_call");
    expect(toolEvent?.type).toBe("tool_call");
    if (toolEvent?.type !== "tool_call") throw new Error("expected tool_call event");
    expect(toolEvent.toolCall.name).toBe("exec");
    expect(toolEvent.toolCall.rawInput).toMatch(/^text\(/);
    expect(toolEvent.toolCall.rawInput).toContain("CONTRACT VIOLATION");
    expect(toolEvent.toolCall.rawInput).not.toContain("fileText");
    expect(run.cancel).toHaveBeenCalled();
  });
});
