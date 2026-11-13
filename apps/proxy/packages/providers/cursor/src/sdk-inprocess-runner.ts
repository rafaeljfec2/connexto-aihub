import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  Agent,
  type AgentOptions,
  type RunResult,
  type RunStatus,
  type SDKImage,
  type SDKMessage,
  type SDKUserMessage,
} from "@cursor/sdk";
import { isRecord } from "@agent-runtime-gateway/compatibility-openai";
import type { CursorToolCall } from "@agent-runtime-gateway/compatibility-openai";
import type {
  AgentImageInput,
  CursorClientToolSpec,
} from "@agent-runtime-gateway/core";
import {
  type ClientToolCallbackBody,
  type ClientToolCallbackServer,
  startClientToolCallbackServer,
} from "./client-mcp/callback-server.js";
import { clientForwardingMcpServers } from "./client-mcp/forwarding.js";
import { attachGrammarInput } from "./client-mcp/grammar-input.js";
import {
  mcpToolCallToClientTool,
  resolveInventoryToolName,
} from "./client-mcp/normalize.js";
import { edgeOwnedBridgePromptPrefix } from "./client-mcp/prompt.js";
import { cursorModelSelection } from "./model-selection.js";
import { CursorProviderError } from "./provider-error.js";
import type { CursorTextEvent } from "./engine/types.js";
import type { CursorSdkCompletion } from "./sdk/types.js";

interface CursorSdkRun {
  readonly id: string;
  readonly status: RunStatus;
  stream(): AsyncGenerator<SDKMessage>;
  wait(): Promise<RunResult>;
  cancel(): Promise<void>;
}

export interface CursorSdkSendOptions {
  readonly mcpServers?: Record<string, unknown>;
}

interface CursorSdkAgent {
  readonly agentId: string;
  send(message: string | SDKUserMessage, options?: CursorSdkSendOptions): Promise<CursorSdkRun>;
  close(): void;
}

export interface CursorSdkInProcessDependencies {
  createAgent(options: AgentOptions): Promise<CursorSdkAgent>;
  deleteAgent(agentId: string): Promise<void>;
}

export interface CursorSdkInProcessInput {
  readonly apiKey: string;
  readonly model: string;
  readonly agentName: string;
  readonly prompt: string;
  readonly workingDirectory?: string;
  readonly clientTools: readonly CursorClientToolSpec[];
  readonly images?: readonly AgentImageInput[];
  readonly signal?: AbortSignal;
}

interface ClientToolInventory {
  readonly clientTools: readonly CursorClientToolSpec[];
  tryCapture(call: CursorToolCall): boolean;
  peekCaptured(): CursorToolCall | null;
  waitForCapture(): Promise<CursorToolCall>;
}

interface CursorSdkAgentResources {
  start(this: void): Promise<{ agent: CursorSdkAgent; run: CursorSdkRun }>;
  cleanup(this: void): Promise<void>;
}

const defaultDependencies: CursorSdkInProcessDependencies = {
  createAgent: createIsolatedAgent,
  deleteAgent: deleteIsolatedAgent,
};

const agentWorkspaces = new Map<string, string>();

export async function createCursorSdkInProcessCompletion(
  input: CursorSdkInProcessInput,
  dependencies: CursorSdkInProcessDependencies = defaultDependencies,
): Promise<CursorSdkCompletion> {
  input.signal?.throwIfAborted();
  const inventory = createClientToolInventory(input.clientTools);
  const resources = createSdkAgentResources(input, inventory, dependencies);

  try {
    const { agent, run } = await resources.start();
    return {
      agentId: agent.agentId,
      runId: run.id,
      stream: streamSdkRun(run, inventory, input, resources.cleanup),
      cancel: resources.cleanup,
    };
  } catch (error) {
    await resources.cleanup();
    throw error;
  }
}

function createSdkAgentResources(
  input: CursorSdkInProcessInput,
  inventory: ClientToolInventory,
  dependencies: CursorSdkInProcessDependencies,
): CursorSdkAgentResources {
  let agent: CursorSdkAgent | undefined;
  let run: CursorSdkRun | undefined;
  let callbackServer: ClientToolCallbackServer | undefined;
  let agentClosed = false;
  let cleanupPromise: Promise<void> | undefined;
  const closeAgent = (): void => {
    if (agent === undefined || agentClosed) return;
    agentClosed = true;
    agent.close();
  };
  const cleanup = function cleanupAgentRun(): Promise<void> {
    if (cleanupPromise === undefined) {
      if (abortHandler !== undefined) {
        input.signal?.removeEventListener("abort", abortHandler);
      }
      cleanupPromise = disposeAgentRun(
        agent,
        run,
        callbackServer,
        dependencies,
        closeAgent,
      );
    }
    return cleanupPromise;
  };
  const abortHandler = (): void => {
    if (run !== undefined) {
      void cleanup().catch(() => undefined);
      return;
    }
    closeAgent();
  };
  input.signal?.addEventListener("abort", abortHandler, { once: true });

  const start = async (): Promise<{ agent: CursorSdkAgent; run: CursorSdkRun }> => {
    if (input.clientTools.length > 0) {
      callbackServer = await startClientToolCallbackServer((body) => {
        const call = callbackBodyToClientTool(body, input.clientTools);
        if (!call) return false;
        return inventory.tryCapture(call);
      });
    }
    agent = await dependencies.createAgent(createAgentOptions(input));
    input.signal?.throwIfAborted();
    const sendOptions =
      input.clientTools.length > 0 && callbackServer !== undefined
        ? {
            mcpServers: clientForwardingMcpServers(input.clientTools, {
              callbackUrl: callbackServer.url,
              callbackToken: callbackServer.token,
            }),
          }
        : undefined;
    run = await agent.send(createUserMessage(input), sendOptions);
    input.signal?.throwIfAborted();
    return { agent, run };
  };

  return { start, cleanup };
}

function createAgentOptions(input: CursorSdkInProcessInput): AgentOptions {
  const hasClientTools = input.clientTools.length > 0;
  return {
    apiKey: input.apiKey,
    model: cursorModelSelection(input.model),
    name: input.agentName,
    mode: "agent",
    tools: hasClientTools ? ["mcp"] : [],
    mcpServers: {},
    local: {
      settingSources: [],
    },
  };
}

async function createIsolatedAgent(options: AgentOptions): Promise<CursorSdkAgent> {
  const workspace = await mkdtemp(join(tmpdir(), "cursor-edge-owned-"));
  try {
    const agent = await Agent.create({
      ...options,
      local: { ...options.local, cwd: workspace },
    });
    agentWorkspaces.set(agent.agentId, workspace);
    return {
      agentId: agent.agentId,
      send: (message, sendOptions) =>
        agent.send(
          message,
          sendOptions as NonNullable<Parameters<typeof agent.send>[1]>,
        ),
      close: () => agent.close(),
    };
  } catch (error) {
    await rm(workspace, { recursive: true, force: true });
    throw error;
  }
}

async function deleteIsolatedAgent(agentId: string): Promise<void> {
  const workspace = agentWorkspaces.get(agentId);
  if (workspace === undefined) return;
  agentWorkspaces.delete(agentId);
  try {
    await Agent.delete(agentId, { cwd: workspace });
  } finally {
    await rm(workspace, { recursive: true, force: true });
  }
}

async function disposeAgentRun(
  agent: CursorSdkAgent | undefined,
  run: CursorSdkRun | undefined,
  callbackServer: ClientToolCallbackServer | undefined,
  dependencies: CursorSdkInProcessDependencies,
  closeAgent: () => void,
): Promise<void> {
  try {
    if (callbackServer !== undefined) await callbackServer.close();
  } catch {
    // Best-effort shutdown for the ephemeral callback server.
  }
  try {
    if (run?.status === "running") await run.cancel();
  } finally {
    if (agent !== undefined) {
      try {
        closeAgent();
      } finally {
        await dependencies.deleteAgent(agent.agentId);
      }
    }
  }
}

function createUserMessage(input: CursorSdkInProcessInput): SDKUserMessage {
  const workspaceContext = input.workingDirectory
    ? `\n\nIDE workspace for client-side tools only: ${JSON.stringify(input.workingDirectory)}`
    : "";
  const toolPolicy = input.clientTools.length
    ? "\n\nRequest one client tool at a time. Wait for its result in a later request before deciding whether another client tool is needed."
    : "";
  const body = edgeOwnedBridgePromptPrefix(input.prompt, input.clientTools);
  const images: SDKImage[] = (input.images ?? []).map((image) => ({
    data: image.data,
    mimeType: image.mimeType,
  }));
  return {
    text: `${body}${workspaceContext}${toolPolicy}`,
    ...(images.length > 0 ? { images } : {}),
  };
}

function createClientToolInventory(
  clientTools: readonly CursorClientToolSpec[],
): ClientToolInventory {
  const names = new Set(
    clientTools.map((tool) => tool.name.trim()).filter((name) => name.length > 0),
  );
  let captured: CursorToolCall | null = null;
  let notifyCapture: ((call: CursorToolCall) => void) | undefined;
  const capturedPromise = new Promise<CursorToolCall>((resolve) => {
    notifyCapture = resolve;
  });
  return {
    clientTools,
    tryCapture(call: CursorToolCall): boolean {
      if (captured !== null || !names.has(call.name)) return false;
      captured = call;
      notifyCapture?.(call);
      return true;
    },
    peekCaptured(): CursorToolCall | null {
      return captured;
    },
    waitForCapture(): Promise<CursorToolCall> {
      return capturedPromise;
    },
  };
}

function callbackBodyToClientTool(
  body: ClientToolCallbackBody,
  clientTools: readonly CursorClientToolSpec[],
): CursorToolCall | null {
  const resolved = resolveInventoryToolName(body.toolName, clientTools);
  if (!resolved) return null;
  return attachGrammarInput(
    { name: resolved, arguments: { ...body.arguments } },
    clientTools,
  );
}

async function* streamSdkRun(
  run: CursorSdkRun,
  inventory: ClientToolInventory,
  input: CursorSdkInProcessInput,
  cleanup: () => Promise<void>,
): AsyncGenerator<CursorTextEvent> {
  let finalText = "";
  const iterator = run.stream()[Symbol.asyncIterator]();
  let pendingNext = iterator.next();

  try {
    while (true) {
      if (input.signal?.aborted) return;
      const alreadyCaptured = inventory.peekCaptured();
      if (alreadyCaptured) {
        await cleanup();
        yield* forwardedToolCallEvents([alreadyCaptured], finalText);
        return;
      }

      const winner = await Promise.race([
        pendingNext.then((result) => ({ kind: "message" as const, result })),
        inventory.waitForCapture().then((call) => ({ kind: "capture" as const, call })),
      ]);

      if (winner.kind === "capture") {
        await cleanup();
        yield* forwardedToolCallEvents([winner.call], finalText);
        return;
      }
      if (winner.result.done) break;
      pendingNext = iterator.next();

      const message = winner.result.value;
      const text = assistantText(message);
      if (text) {
        finalText += text;
        yield { type: "text", text };
      }
      const toolCall = firstClientToolCallFromMessage(message, inventory);
      if (toolCall) {
        await cleanup();
        yield* forwardedToolCallEvents([toolCall], finalText);
        return;
      }
    }

    yield* completedSdkRunEvents(run, finalText, cleanup);
  } finally {
    await cleanup();
  }
}

function firstClientToolCallFromMessage(
  message: SDKMessage,
  inventory: ClientToolInventory,
): CursorToolCall | null {
  const fromStream = clientToolCallsFromMessage(message, inventory);
  if (fromStream.length > 0) return fromStream[0] ?? null;
  return null;
}

function clientToolCallsFromMessage(
  message: SDKMessage,
  inventory: ClientToolInventory,
): CursorToolCall[] {
  if (message.type === "assistant") {
    return clientToolCallsFromAssistantMessage(message, inventory);
  }
  if (message.type !== "tool_call" || message.status !== "running") return [];
  const call = clientToolCallFromMcp(message.name, message.args, inventory);
  return call ? [call] : [];
}

function* forwardedToolCallEvents(
  toolCalls: CursorToolCall[],
  finalText: string,
): Generator<CursorTextEvent> {
  for (const toolCall of toolCalls) yield { type: "tool_call", toolCall };
  yield { type: "done", finalText, toolCalls };
}

async function* completedSdkRunEvents(
  run: CursorSdkRun,
  finalText: string,
  cleanup: () => Promise<void>,
): AsyncGenerator<CursorTextEvent> {
  const result = await run.wait();
  if (result.status !== "finished") {
    throw new Error(
      result.error?.message ?? `Cursor SDK run ended with ${result.status}.`,
    );
  }
  await cleanup();
  yield { type: "done", finalText, toolCalls: [] };
}

function assistantText(message: SDKMessage): string {
  if (message.type !== "assistant") return "";
  return message.message.content
    .flatMap((block) => (block.type === "text" ? [block.text] : []))
    .join("");
}

function clientToolCallsFromAssistantMessage(
  message: SDKMessage,
  inventory: ClientToolInventory,
): CursorToolCall[] {
  if (message.type !== "assistant") return [];
  return message.message.content.flatMap((block) =>
    block.type === "tool_use"
      ? (() => {
          const call = clientToolCallFromMcp(block.name, block.input, inventory);
          return call ? [call] : [];
        })()
      : [],
  );
}

function clientToolCallFromMcp(
  name: string,
  argsValue: unknown,
  inventory: ClientToolInventory,
): CursorToolCall | null {
  if (name !== "mcp") {
    throw invalidClientTool(
      "Cursor SDK requested a tool outside the client tool inventory.",
    );
  }
  const call = mcpToolCallToClientTool(argsValue, inventory.clientTools);
  if (!call) {
    throw invalidClientTool(
      "Cursor SDK requested a tool outside the client tool inventory.",
    );
  }
  inventory.tryCapture(call);
  return call;
}

function invalidClientTool(message: string): CursorProviderError {
  return new CursorProviderError(message, "cursor_invalid_client_tool");
}
