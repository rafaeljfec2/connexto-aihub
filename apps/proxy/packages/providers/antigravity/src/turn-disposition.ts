import type { AgentToolChoice, AgentToolSpec } from "@agent-runtime-gateway/core";

export interface RecordedToolCall {
  readonly id: string;
  readonly name: string;
  readonly arguments: string;
}

export interface ToolTurnDecision {
  readonly status: "completed" | "cancelled" | "failed";
  readonly reason?: string;
}

/** A delegated client tool ends the HTTP turn as a normal OpenAI tool call, not a cancel. */
export function acpTurnDisposition(input: {
  readonly userCancelled: boolean;
  readonly stopReason: string | undefined;
  readonly toolCallCount: number;
}): "completed" | "cancelled" {
  if (input.userCancelled) return "cancelled";
  if (input.stopReason === "cancelled" && input.toolCallCount === 0) return "cancelled";
  return "completed";
}

export function assertRequestedToolIsDeclared(
  toolChoice: AgentToolChoice | undefined,
  tools: readonly AgentToolSpec[] | undefined,
): void {
  const requiredName = functionToolChoiceName(toolChoice);
  if (requiredName === undefined) return;
  const toolExists = tools?.some((tool) => tool.name === requiredName) === true;
  if (!toolExists) {
    throw new Error(
      `Requested tool '${requiredName}' in tool_choice was not found in declared tools.`,
    );
  }
}

export function requiredToolChoiceFailure(
  toolChoice: AgentToolChoice | undefined,
  recordedToolNames: readonly string[],
): string | undefined {
  if (toolChoice === "required" && recordedToolNames.length === 0) {
    return "Model failed to call any tool despite tool_choice='required'.";
  }
  const requiredName = functionToolChoiceName(toolChoice);
  if (requiredName !== undefined && !recordedToolNames.includes(requiredName)) {
    return `Model failed to call required tool '${requiredName}' despite tool_choice directive.`;
  }
  return undefined;
}

/** Decision for the race winner that ends the turn on a delegated client tool. */
export function delegatedToolTurnDecision(input: {
  readonly userCancelled: boolean;
  readonly toolChoice: AgentToolChoice | undefined;
  readonly recordedToolNames: readonly string[];
}): ToolTurnDecision {
  return decideRecordedToolTurn({
    userCancelled: input.userCancelled,
    intentionalDelegatedStop: true,
    promptFailed: false,
    toolChoice: input.toolChoice,
    recordedToolNames: input.recordedToolNames,
  });
}

/**
 * Catch-path outcome. Recorded tools complete the turn only for the intentional
 * delegated stop, and only when tool_choice is satisfied. User abort stays
 * cancelled. A required-tool miss or an unrelated prompt failure stays failed.
 */
export function caughtTurnOutcome(input: {
  readonly userCancelled: boolean;
  readonly intentionalDelegatedStop: boolean;
  readonly toolChoice: AgentToolChoice | undefined;
  readonly recordedToolNames: readonly string[];
}): "completed" | "cancelled" | "failed" {
  return decideRecordedToolTurn({
    userCancelled: input.userCancelled,
    intentionalDelegatedStop: input.intentionalDelegatedStop,
    promptFailed: true,
    toolChoice: input.toolChoice,
    recordedToolNames: input.recordedToolNames,
  }).status;
}

function decideRecordedToolTurn(input: {
  readonly userCancelled: boolean;
  readonly intentionalDelegatedStop: boolean;
  readonly promptFailed: boolean;
  readonly toolChoice: AgentToolChoice | undefined;
  readonly recordedToolNames: readonly string[];
}): ToolTurnDecision {
  if (input.userCancelled) return { status: "cancelled" };
  const reason = requiredToolChoiceFailure(input.toolChoice, input.recordedToolNames);
  if (reason !== undefined) return { status: "failed", reason };
  if (input.promptFailed && !input.intentionalDelegatedStop) return { status: "failed" };
  return { status: "completed" };
}

function functionToolChoiceName(toolChoice: AgentToolChoice | undefined): string | undefined {
  if (typeof toolChoice === "object" && toolChoice !== null && toolChoice.type === "function") {
    return toolChoice.function.name;
  }
  return undefined;
}
