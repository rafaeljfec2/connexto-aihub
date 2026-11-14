import type {
  AgentReasoningEffort,
  AgentToolChoice,
} from "@agent-runtime-gateway/core";
import { isRecord, type PreparedRequest } from "@agent-runtime-gateway/compatibility-openai";
import type { GatewayServices } from "../../services/gateway.js";
import type { ProviderExecutionTarget } from "../../utils/provider-target.js";

export interface ResponseDeliveryContext {
  readonly id: string;
  readonly created: number;
  readonly requestId: string;
  readonly principal: string;
  readonly sessionAffinity: string;
  readonly target: ProviderExecutionTarget;
  readonly prepared: PreparedRequest;
  readonly providerStart: number;
  readonly services: GatewayServices;
  readonly signal: AbortSignal;
  readonly textFormat?: unknown;
  readonly reasoningEffort?: string;
}

/** Reads optional reasoning effort from OpenAI Responses request body fields. */
export function readReasoningEffort(body: Record<string, unknown>): string | undefined {
  const nestedEffort = isRecord(body.reasoning) ? body.reasoning.effort : undefined;
  const candidate = nestedEffort ?? body.reasoning_effort;
  return typeof candidate === "string" ? candidate : undefined;
}

/** Narrows reasoning effort string to the executor's supported union. */
export function toExecutorReasoningEffort(
  value: string | undefined,
): AgentReasoningEffort | undefined {
  return value === "low" || value === "medium" || value === "high" ? value : undefined;
}

/** Narrows OpenAI tool_choice payloads into the core AgentToolChoice union. */
export function readToolChoice(value: unknown): AgentToolChoice | undefined {
  if (value === "none" || value === "auto" || value === "required") {
    return value;
  }
  if (!isRecord(value) || value.type !== "function") {
    return undefined;
  }
  if (isRecord(value.function) && typeof value.function.name === "string") {
    return { type: "function", function: { name: value.function.name } };
  }
  if (typeof value.name === "string") {
    return { type: "function", function: { name: value.name } };
  }
  return undefined;
}

function isClientToolCallItem(item: Record<string, unknown>): boolean {
  return item.type === "function_call" || item.type === "custom_tool_call";
}

function isClientToolOutputItem(item: Record<string, unknown>): boolean {
  return item.type === "function_call_output" || item.type === "custom_tool_call_output";
}

/** Extracts tool output items following the most recent function or custom tool call. */
export function extractActiveResponseToolResults(
  inputItems: readonly unknown[],
): { toolCallId: string; content: string }[] {
  let lastFunctionCallIdx = -1;
  for (let i = inputItems.length - 1; i >= 0; i--) {
    const item = inputItems[i];
    if (isRecord(item) && isClientToolCallItem(item)) {
      lastFunctionCallIdx = i;
      break;
    }
  }

  const activeItems =
    lastFunctionCallIdx >= 0 ? inputItems.slice(lastFunctionCallIdx + 1) : inputItems;

  return activeItems
    .filter(
      (item): item is Record<string, unknown> =>
        isRecord(item) && isClientToolOutputItem(item),
    )
    .map((item) => ({
      toolCallId: typeof item.call_id === "string" ? item.call_id : "",
      content:
        typeof item.output === "string"
          ? item.output
          : JSON.stringify(item.output ?? ""),
    }));
}
