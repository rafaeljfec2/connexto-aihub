import type { CursorToolCall } from "@agent-runtime-gateway/compatibility-openai";
import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import type { SDKMessage } from "@cursor/sdk";
import { enforceClientToolContract } from "./client-mcp/client-tool-contract.js";
import {
  mcpToolCallToClientTool,
} from "./client-mcp/normalize.js";
import type { CursorTextEvent } from "./engine/types.js";
import { CursorProviderError } from "./provider-error.js";
import type { CursorSdkInProcessInput } from "./sdk-inprocess-types.js";

export interface CursorSdkRun {
  readonly id: string;
  readonly status: import("@cursor/sdk").RunStatus;
  stream(): AsyncGenerator<SDKMessage>;
  wait(): Promise<import("@cursor/sdk").RunResult>;
  cancel(): Promise<void>;
}

interface ClientToolInventory {
  readonly clientTools: readonly CursorClientToolSpec[];
  tryCapture(call: CursorToolCall): boolean;
  peekCaptured(): CursorToolCall | null;
  waitForCapture(): Promise<CursorToolCall>;
}

export async function* streamSdkRun(
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
        yield* forwardedToolCallEvents([alreadyCaptured], finalText, input.clientTools);
        await bestEffortCleanup(cleanup);
        return;
      }

      const winner = await Promise.race([
        pendingNext.then((result) => ({ kind: "message" as const, result })),
        inventory.waitForCapture().then((call) => ({ kind: "capture" as const, call })),
      ]);

      if (winner.kind === "capture") {
        yield* forwardedToolCallEvents([winner.call], finalText, input.clientTools);
        await bestEffortCleanup(cleanup);
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
        yield* forwardedToolCallEvents([toolCall], finalText, input.clientTools);
        await bestEffortCleanup(cleanup);
        return;
      }
    }

    yield* completedSdkRunEvents(run, finalText, cleanup);
  } finally {
    await bestEffortCleanup(cleanup);
  }
}

async function bestEffortCleanup(cleanup: () => Promise<void>): Promise<void> {
  try {
    await cleanup();
  } catch {
    // SDK cancel after a forwarded client tool is best-effort only.
  }
}

function* forwardedToolCallEvents(
  toolCalls: CursorToolCall[],
  finalText: string,
  clientTools: readonly CursorClientToolSpec[],
): Generator<CursorTextEvent> {
  const enforced = toolCalls.map((toolCall) =>
    enforceClientToolContract(toolCall, clientTools),
  );
  for (const toolCall of enforced) yield { type: "tool_call", toolCall };
  yield { type: "done", finalText, toolCalls: enforced };
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
  yield { type: "done", finalText, toolCalls: [] };
  await bestEffortCleanup(cleanup);
}

function assistantText(message: SDKMessage): string {
  if (message.type !== "assistant") return "";
  return message.message.content
    .flatMap((block) => (block.type === "text" ? [block.text] : []))
    .join("");
}

function firstClientToolCallFromMessage(
  message: SDKMessage,
  inventory: ClientToolInventory,
): CursorToolCall | null {
  const fromStream = clientToolCallsFromMessage(message, inventory);
  if (fromStream.length > 0) return fromStream[0] ?? null;
  return null;
}

function attachSdkCallId(
  call: CursorToolCall,
  sdkCallId: string | undefined,
): CursorToolCall {
  const trimmed = sdkCallId?.trim();
  if (!trimmed) return call;
  return { ...call, sdkCallId: trimmed };
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
  return call ? [attachSdkCallId(call, message.call_id)] : [];
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
          return call ? [attachSdkCallId(call, block.id)] : [];
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
  const enforced = enforceClientToolContract(call, inventory.clientTools);
  inventory.tryCapture(enforced);
  return enforced;
}

function invalidClientTool(message: string): CursorProviderError {
  return new CursorProviderError(message, "cursor_invalid_client_tool");
}
