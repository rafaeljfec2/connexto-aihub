import type { AgentOptions, RunResult, RunStatus, SDKMessage } from "@cursor/sdk";
import { describe, expect, it, vi } from "vitest";
import { CLIENT_MCP_SERVER_NAME } from "../src/client-mcp/constants.js";
import {
  createCursorSdkInProcessCompletion,
  type CursorSdkInProcessDependencies,
  type CursorSdkSendOptions,
} from "../src/sdk-inprocess-runner.js";
import type { CursorTextEvent } from "../src/engine/types.js";

describe("createCursorSdkInProcessCompletion", () => {
  it("returns client-owned tool calls and cancels the SDK run before resolving the callback", async () => {
    let options: AgentOptions | undefined;
    let sendOptions: CursorSdkSendOptions | undefined;
    let runStatus: RunStatus = "running";
    const toolCallMessage: SDKMessage = {
      type: "tool_call",
      agent_id: "agent-1",
      run_id: "run-1",
      call_id: "call-1",
      name: "mcp",
      status: "running",
      args: {
        providerIdentifier: "client",
        toolName: "write",
        args: { path: "note.txt", fileText: "hello" },
      },
    };
    const run = {
      id: "run-1",
      get status() {
        return runStatus;
      },
      async *stream(): AsyncGenerator<SDKMessage> {
        yield toolCallMessage;
      },
      wait(): Promise<RunResult> {
        return Promise.resolve({
          id: "run-1",
          status: runStatus === "running" ? "cancelled" : runStatus,
        });
      },
      cancel: vi.fn(() => {
        runStatus = "cancelled";
        return Promise.resolve();
      }),
    };
    const closeAgent = vi.fn();
    const agent = {
      agentId: "agent-1",
      send: vi.fn((_message, opts?: CursorSdkSendOptions) => {
        sendOptions = opts;
        return Promise.resolve(run);
      }),
      close: closeAgent,
    };
    const deleteAgent = vi.fn(() => Promise.resolve());
    const dependencies: CursorSdkInProcessDependencies = {
      createAgent: vi.fn((agentOptions) => {
        options = agentOptions;
        return Promise.resolve(agent);
      }),
      deleteAgent,
    };

    const completion = await createCursorSdkInProcessCompletion(
      {
        apiKey: "cursor-key",
        model: "composer-2.5",
        agentName: "gateway-test",
        prompt: "Create note.txt",
        workingDirectory: "/ide/project",
        clientTools: [
          {
            name: "write",
            description: "Write a client-owned file",
            parameters: { type: "object", properties: { path: { type: "string" } } },
          },
        ],
      },
      dependencies,
    );

    const events: CursorTextEvent[] = [];
    for await (const event of completion.stream) events.push(event);

    expect(options?.tools).toEqual(["mcp"]);
    expect(options?.local?.customTools).toBeUndefined();
    expect(sendOptions?.mcpServers?.[CLIENT_MCP_SERVER_NAME]).toMatchObject({ type: "stdio" });
    expect(options?.local?.settingSources).toEqual([]);
    expect(options?.local?.cwd).not.toBe("/ide/project");
    expect(agent.send.mock.calls[0]?.[0]).toMatchObject({
      text: expect.stringContaining('providerIdentifier "client"'),
    });
    const sentText = String(
      (agent.send.mock.calls[0]?.[0] as { text?: string } | undefined)?.text ?? "",
    );
    expect(sentText).toContain("Create note.txt");
    expect(sentText.indexOf("GetDynamicTools")).toBeGreaterThan(sentText.indexOf("Create note.txt"));
    expect(sentText).toContain('"toolName":"write"');
    expect(run.cancel).toHaveBeenCalled();
    expect(events).toContainEqual({
      type: "tool_call",
      toolCall: {
        name: "write",
        arguments: { path: "note.txt", fileText: "hello" },
        sdkCallId: "call-1",
      },
    });
    expect(events.at(-1)).toEqual({
      type: "done",
      finalText: "",
      toolCalls: [
        {
          name: "write",
          arguments: { path: "note.txt", fileText: "hello" },
          sdkCallId: "call-1",
        },
      ],
    });
    expect(closeAgent).toHaveBeenCalledOnce();
    expect(deleteAgent).toHaveBeenCalledWith("agent-1");
  });

  it("disables all tools for an ask request without a client tool inventory", async () => {
    let options: AgentOptions | undefined;
    let runStatus: RunStatus = "finished";
    const run = {
      id: "run-2",
      get status() {
        return runStatus;
      },
      async *stream(): AsyncGenerator<SDKMessage> {
        await Promise.resolve();
        yield {
          type: "assistant",
          agent_id: "agent-2",
          run_id: "run-2",
          message: { role: "assistant", content: [{ type: "text", text: "Hello" }] },
        };
      },
      wait(): Promise<RunResult> {
        return Promise.resolve({
          id: "run-2",
          status: runStatus === "running" ? "cancelled" : runStatus,
          result: "Hello",
        });
      },
      cancel: vi.fn(() => {
        runStatus = "cancelled";
        return Promise.resolve();
      }),
    };
    const deleteAgent = vi.fn(() => Promise.resolve());
    const agent = {
      agentId: "agent-2",
      send: vi.fn(() => Promise.resolve(run)),
      close: vi.fn(),
    };
    const dependencies: CursorSdkInProcessDependencies = {
      createAgent: vi.fn((agentOptions) => {
        options = agentOptions;
        return Promise.resolve(agent);
      }),
      deleteAgent,
    };

    const completion = await createCursorSdkInProcessCompletion(
      {
        apiKey: "cursor-key",
        model: "composer-2.5",
        agentName: "gateway-test",
        prompt: "Say hello",
        clientTools: [],
      },
      dependencies,
    );
    const events: CursorTextEvent[] = [];
    for await (const event of completion.stream) events.push(event);

    expect(options?.tools).toEqual([]);
    expect(options?.local?.customTools).toBeUndefined();
    expect(events).toContainEqual({ type: "text", text: "Hello" });
    expect(events.at(-1)).toEqual({ type: "done", finalText: "Hello", toolCalls: [] });
    expect(run.cancel).not.toHaveBeenCalled();
    expect(deleteAgent).toHaveBeenCalledWith("agent-2");
  });

  it("cancels and cleans up an agent when the stream is never consumed", async () => {
    const run = {
      id: "run-3",
      status: "running" as const,
      stream: vi.fn(),
      wait: vi.fn(),
      cancel: vi.fn(() => Promise.resolve()),
    };
    const closeAgent = vi.fn();
    const deleteAgent = vi.fn(() => Promise.resolve());
    const dependencies: CursorSdkInProcessDependencies = {
      createAgent: vi.fn(() =>
        Promise.resolve({
          agentId: "agent-3",
          send: () => Promise.resolve(run),
          close: closeAgent,
        }),
      ),
      deleteAgent,
    };

    const completion = await createCursorSdkInProcessCompletion(
      {
        apiKey: "cursor-key",
        model: "composer-2.5",
        agentName: "gateway-test",
        prompt: "Say hello",
        clientTools: [],
      },
      dependencies,
    );

    await completion.cancel?.();

    expect(run.stream).not.toHaveBeenCalled();
    expect(run.cancel).toHaveBeenCalledOnce();
    expect(closeAgent).toHaveBeenCalledOnce();
    expect(deleteAgent).toHaveBeenCalledWith("agent-3");
  });

  it("closes a pending SDK send and cancels its run after request abort", async () => {
    const controller = new AbortController();
    let runStatus: RunStatus = "running";
    const run = {
      id: "run-5",
      get status() {
        return runStatus;
      },
      async *stream(): AsyncGenerator<SDKMessage> {
        await Promise.resolve();
        yield {
          type: "status",
          agent_id: "agent-5",
          run_id: "run-5",
          status: "RUNNING",
        };
      },
      wait: vi.fn(() => Promise.resolve({ id: "run-5", status: "cancelled" as const })),
      cancel: vi.fn(() => {
        runStatus = "cancelled";
        return Promise.resolve();
      }),
    };
    let resolveRun: ((value: typeof run) => void) | undefined;
    const closeAgent = vi.fn();
    const deleteAgent = vi.fn(() => Promise.resolve());
    const agent = {
      agentId: "agent-5",
      send: vi.fn(
        () =>
          new Promise<typeof run>((resolve) => {
            resolveRun = resolve;
          }),
      ),
      close: closeAgent,
    };
    const dependencies: CursorSdkInProcessDependencies = {
      createAgent: vi.fn(() => Promise.resolve(agent)),
      deleteAgent,
    };

    const completionPromise = createCursorSdkInProcessCompletion(
      {
        apiKey: "cursor-key",
        model: "composer-2.5",
        agentName: "gateway-test",
        prompt: "Say hello",
        clientTools: [],
        signal: controller.signal,
      },
      dependencies,
    );
    await vi.waitFor(() => expect(agent.send).toHaveBeenCalledOnce());
    controller.abort();
    expect(closeAgent).toHaveBeenCalledOnce();
    resolveRun?.(run);

    await expect(completionPromise).rejects.toThrow();
    expect(run.cancel).toHaveBeenCalledOnce();
    expect(deleteAgent).toHaveBeenCalledWith("agent-5");
  });

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

  it("does not forward a client tool when SDK cancel fails during cleanup", async () => {
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

    await expect(async () => {
      for await (const _event of completion.stream) {
        // drain
      }
    }).rejects.toThrow("cancel failed");
    expect(run.cancel).toHaveBeenCalled();
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
        clientTools: [{ name: "exec" }],
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
        arguments: { code: "return 1" },
      }),
    });
    expect(response.ok).toBe(true);

    const events = await eventsPromise;
    expect(events).toContainEqual({
      type: "tool_call",
      toolCall: { name: "exec", arguments: { code: "return 1" } },
    });
    expect(run.cancel).toHaveBeenCalled();
  });
});
