import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { Agent, type AgentOptions, type SDKUserMessage } from "@cursor/sdk";
import type { CursorToolCall } from "@agent-runtime-gateway/compatibility-openai";
import {
  type ClientToolCallbackBody,
  type ClientToolCallbackServer,
  startClientToolCallbackServer,
} from "./client-mcp/callback-server.js";
import { clientForwardingMcpServers } from "./client-mcp/forwarding.js";
import { enforceClientToolContract } from "./client-mcp/client-tool-contract.js";
import { attachGrammarInput } from "./client-mcp/grammar-input.js";
import { resolveInventoryToolName } from "./client-mcp/normalize.js";
import { edgeOwnedBridgePromptPrefix } from "./client-mcp/prompt.js";
import { cursorModelSelection } from "./model-selection.js";
import type { CursorSdkRun } from "./sdk-inprocess-stream.js";
import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import type {
  CursorSdkInProcessDependencies,
  CursorSdkInProcessInput,
  CursorSdkSendOptions,
} from "./sdk-inprocess-types.js";

interface CursorSdkAgent {
  readonly agentId: string;
  send(message: string | SDKUserMessage, options?: CursorSdkSendOptions): Promise<CursorSdkRun>;
  close(): void;
}

interface ClientToolInventory {
  readonly clientTools: readonly CursorClientToolSpec[];
  tryCapture(call: CursorToolCall): boolean;
}

interface CursorSdkAgentResources {
  start(this: void): Promise<{ agent: CursorSdkAgent; run: CursorSdkRun }>;
  cleanup(this: void): Promise<void>;
}

const agentWorkspaces = new Map<string, string>();

export function createSdkAgentResources(
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
        const enforced = enforceClientToolContract(call, input.clientTools);
        return inventory.tryCapture(enforced);
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

export async function createIsolatedAgent(options: AgentOptions): Promise<CursorSdkAgent> {
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

export async function deleteIsolatedAgent(agentId: string): Promise<void> {
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
    ? "\n\nBefore each client tool, tell the user in 1-3 sentences what you are doing and which tool comes next. Request one client tool at a time and wait for its result before choosing the next step."
    : "";
  const body = edgeOwnedBridgePromptPrefix(input.prompt, input.clientTools);
  const images = (input.images ?? []).map((image) => ({
    data: image.data,
    mimeType: image.mimeType,
  }));
  return {
    text: `${body}${workspaceContext}${toolPolicy}`,
    ...(images.length > 0 ? { images } : {}),
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
