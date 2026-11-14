import type { AgentExecutionInput, AgentEvent } from "@agent-runtime-gateway/core";
import { AsyncEventQueue } from "./async-event-queue.js";
import { AntigravityMcpBridge } from "./mcp/index.js";
import { delegatedToolCallPayload } from "./mcp/openai-tool-shape.js";
import type { McpBridgeToolCallRequest, McpBridgeToolCallResult } from "./mcp/types.js";
import type { PendingToolRegistry } from "./pending-tools.js";
import type { RecordedToolCall } from "./turn-disposition.js";

export interface DelegatedTurnController {
  readonly recordedToolCalls: RecordedToolCall[];
  readonly delegatedTurn: Promise<void>;
  bridge: AntigravityMcpBridge | undefined;
  start(): Promise<{
    readonly bridge: AntigravityMcpBridge | undefined;
    readonly mcpServers: unknown[] | undefined;
  }>;
}

interface DelegatedEnd {
  scheduled: boolean;
  end: () => void;
}

export function createDelegatedTurnController(input: {
  readonly execution: AgentExecutionInput;
  readonly runId: string;
  readonly queue: AsyncEventQueue;
  readonly pendingTools: PendingToolRegistry;
}): DelegatedTurnController {
  const recordedToolCalls: RecordedToolCall[] = [];
  const endState: DelegatedEnd = { scheduled: false, end: () => {} };
  const delegatedTurn = new Promise<void>((resolve) => {
    endState.end = resolve;
  });
  const controller: DelegatedTurnController = {
    recordedToolCalls,
    delegatedTurn,
    bridge: undefined,
    start: () => startDelegatedBridge(input, recordedToolCalls, controller, endState),
  };
  return controller;
}

async function startDelegatedBridge(
  input: {
    readonly execution: AgentExecutionInput;
    readonly runId: string;
    readonly queue: AsyncEventQueue;
    readonly pendingTools: PendingToolRegistry;
  },
  recordedToolCalls: RecordedToolCall[],
  controller: DelegatedTurnController,
  endState: DelegatedEnd,
): Promise<{
  readonly bridge: AntigravityMcpBridge | undefined;
  readonly mcpServers: unknown[] | undefined;
}> {
  const { execution } = input;
  if (!execution.tools || execution.tools.length === 0 || execution.toolChoice === "none") {
    return { bridge: undefined, mcpServers: undefined };
  }
  const bridge = new AntigravityMcpBridge({
    sessionId: execution.sessionId,
    tools: execution.tools,
    allowParallelToolCalls: execution.parallelToolCalls !== false,
    onToolCall: (call) => {
      publishDelegatedToolCall(input, recordedToolCalls, call);
      scheduleDelegatedTurnEnd(endState);
    },
    onToolResult: (callId, toolResult) => {
      publishDelegatedToolResult(input.queue, input.runId, callId, toolResult);
    },
  });
  controller.bridge = bridge;
  const mcpConfig = await bridge.start();
  return { bridge, mcpServers: [mcpConfig] };
}

function scheduleDelegatedTurnEnd(endState: DelegatedEnd): void {
  if (endState.scheduled) return;
  endState.scheduled = true;
  // Finish the HTTP turn after this MCP response is written. Cancelling
  // inside tools/call deadlocks Gemini and leaves Codex on "...".
  setTimeout(() => endState.end(), 0);
}

function publishDelegatedToolCall(
  input: {
    readonly execution: AgentExecutionInput;
    readonly runId: string;
    readonly queue: AsyncEventQueue;
    readonly pendingTools: PendingToolRegistry;
  },
  recordedToolCalls: RecordedToolCall[],
  call: McpBridgeToolCallRequest,
): void {
  input.pendingTools.register({
    callId: call.callId,
    toolName: call.name,
    arguments: call.arguments,
    sessionId: input.execution.sessionId,
  });
  const spec = input.execution.tools?.find((tool) => tool.name === call.name);
  const payload = delegatedToolCallPayload(spec, call.arguments);
  const event: AgentEvent = {
    type: "tool.call",
    runId: input.runId,
    callId: call.callId,
    toolName: call.name,
    arguments: payload.arguments,
    ...(payload.callKind === "custom" ? { callKind: "custom" as const } : {}),
  };
  input.queue.push(event);
  recordedToolCalls.push({
    id: call.callId,
    name: call.name,
    arguments: payload.arguments,
  });
}

function publishDelegatedToolResult(
  queue: AsyncEventQueue,
  runId: string,
  callId: string,
  toolResult: McpBridgeToolCallResult,
): void {
  queue.push({ type: "tool.result", runId, callId, output: toolResult.output });
}
