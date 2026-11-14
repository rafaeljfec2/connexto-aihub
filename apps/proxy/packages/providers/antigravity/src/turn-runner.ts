import type { AgentExecutionInput, AgentExecutionUsage, AgentResult } from "@agent-runtime-gateway/core";
import { DEFAULT_PROMPT_TURN_TIMEOUT_MS } from "./acp/constants.js";
import type { AgentTurnResult, ContentBlock } from "./acp/index.js";
import type { AntigravityAcpClient } from "./acp-client.js";
import { execFollowUpBlocks, shouldRetryExec } from "./exec-follow-up.js";
import { AsyncEventQueue } from "./async-event-queue.js";
import {
  createCancelledResult,
  createCompletedResult,
  createFailedResult,
  createRunStartedEvent,
  createTerminalEvent,
  mapAcpUpdateToEvent,
} from "./events.js";
import type { AntigravityMcpBridge } from "./mcp/index.js";
import { promptBlocksFor } from "./prompt-blocks.js";
import { createRunCancellation } from "./run-cancellation.js";
import {
  acpTurnDisposition,
  caughtTurnOutcome,
  delegatedToolTurnDecision,
  requiredToolChoiceFailure,
  type RecordedToolCall,
} from "./turn-disposition.js";

export interface AntigravityRunInput {
  readonly client: AntigravityAcpClient;
  readonly input: AgentExecutionInput;
  readonly acpSessionId: string;
  readonly runId: string;
  readonly queue: AsyncEventQueue;
  readonly bridge: AntigravityMcpBridge | undefined;
  readonly recordedToolCalls: RecordedToolCall[];
  readonly delegatedTurn: Promise<void>;
  readonly cancelGraceMs: number;
  readonly onSessionReleased: () => void;
}

interface TurnContext {
  readonly client: AntigravityAcpClient;
  readonly input: AgentExecutionInput;
  readonly acpSessionId: string;
  readonly runId: string;
  readonly queue: AsyncEventQueue;
  readonly bridge: AntigravityMcpBridge | undefined;
  readonly recordedToolCalls: RecordedToolCall[];
  readonly delegatedTurn: Promise<void>;
  readonly isCancelRequested: () => boolean;
  readonly getCancellationFailure: () => unknown;
  readonly abortListener: () => void;
  readonly onSessionReleased: () => void;
  readonly promptOverride: { blocks?: readonly ContentBlock[] };
}

interface RacedTurn {
  readonly result: AgentResult;
  readonly stoppedForClientTool: boolean;
}

type SettledPrompt =
  | { readonly ok: true; readonly turn: AgentTurnResult }
  | { readonly ok: false; readonly error: unknown };

export function createAntigravityRun(input: AntigravityRunInput): {
  id: string;
  events: AsyncEventQueue;
  wait: () => Promise<AgentResult>;
  cancel: () => Promise<void>;
} {
  input.queue.push(
    createRunStartedEvent({
      runId: input.runId,
      sessionId: input.input.sessionId,
      model: input.input.model,
    }),
  );
  const cancellation = createRunCancellation(input.cancelGraceMs);
  const resultPromise = runAntigravityTurn({
    client: input.client,
    input: input.input,
    acpSessionId: input.acpSessionId,
    runId: input.runId,
    queue: input.queue,
    bridge: input.bridge,
    recordedToolCalls: input.recordedToolCalls,
    delegatedTurn: input.delegatedTurn,
    isCancelRequested: () => cancellation.isCancelRequested(),
    getCancellationFailure: () => cancellation.getCancellationFailure(),
    abortListener: () => cancellation.abortListener(),
    onSessionReleased: input.onSessionReleased,
    promptOverride: {},
  });
  cancellation.arm({
    client: input.client,
    acpSessionId: input.acpSessionId,
    resultPromise,
    signal: input.input.signal,
  });
  return {
    id: input.runId,
    events: input.queue,
    wait: () => resultPromise,
    cancel: () => cancellation.request(),
  };
}

async function runAntigravityTurn(ctx: TurnContext): Promise<AgentResult> {
  let raced = await resolveTurnRace(ctx);
  if (shouldRetryExec({ stoppedForClientTool: raced.stoppedForClientTool, status: raced.result.status, toolCallCount: ctx.recordedToolCalls.length, userCancelled: ctx.isCancelRequested(), toolCount: ctx.input.tools?.length ?? 0, toolChoice: ctx.input.toolChoice, text: raced.result.output?.text ?? "" })) {
    ctx.promptOverride.blocks = execFollowUpBlocks();
    raced = await resolveTurnRace(ctx);
  }
  const failedCleanup = !raced.stoppedForClientTool && ctx.getCancellationFailure() !== undefined;
  const result = failedCleanup ? createFailedResult() : raced.result;
  if (raced.stoppedForClientTool) return finishDelegatedClientTurn(ctx, result);
  return finishPromptTurn(ctx, result);
}

async function resolveTurnRace(ctx: TurnContext): Promise<RacedTurn> {
  const stop = { intentional: false };
  const promptResult = startPrompt(ctx);
  try {
    const winner = await Promise.race([
      promptResult.then((settled) => ({ kind: "prompt" as const, settled })),
      ctx.delegatedTurn.then(() => ({ kind: "delegated" as const })),
    ]);
    if (winner.kind === "delegated") return applyDelegatedWinner(ctx, stop);
    return applyPromptWinner(ctx, winner.settled);
  } catch {
    return {
      result: resultFromCaughtTurn(ctx, stop.intentional),
      stoppedForClientTool: stop.intentional,
    };
  }
}

function startPrompt(ctx: TurnContext): Promise<SettledPrompt> {
  return ctx.client
    .prompt(ctx.acpSessionId, ctx.promptOverride.blocks ?? promptBlocksFor(ctx.input), {
      timeoutMs: DEFAULT_PROMPT_TURN_TIMEOUT_MS,
      onEvent: (kind, update) => publishPromptEvent(ctx, kind, update),
    })
    .then(
      (turn) => ({ ok: true as const, turn }),
      (error: unknown) => ({ ok: false as const, error }),
    );
}

function publishPromptEvent(ctx: TurnContext, kind: string, update: unknown): void {
  const event = mapAcpUpdateToEvent(ctx.runId, kind, update);
  if (event === undefined) return;
  ctx.queue.push(event);
}

function applyDelegatedWinner(ctx: TurnContext, stop: { intentional: boolean }): RacedTurn {
  const decision = delegatedToolTurnDecision({
    userCancelled: ctx.isCancelRequested(),
    toolChoice: ctx.input.toolChoice,
    recordedToolNames: recordedNames(ctx),
  });
  if (decision.status === "failed") {
    throw new Error(decision.reason ?? "tool_choice was not satisfied.");
  }
  if (decision.status === "cancelled") {
    return { result: createCancelledResult(), stoppedForClientTool: false };
  }
  stop.intentional = true;
  return {
    result: createCompletedResult({ text: "", toolCalls: ctx.recordedToolCalls }),
    stoppedForClientTool: true,
  };
}

function applyPromptWinner(ctx: TurnContext, settled: SettledPrompt): RacedTurn {
  if (!settled.ok) throw settled.error;
  const failure = requiredToolChoiceFailure(ctx.input.toolChoice, recordedNames(ctx));
  if (failure !== undefined) throw new Error(failure);
  const disposition = acpTurnDisposition({
    userCancelled: ctx.isCancelRequested(),
    stopReason: settled.turn.prompt.stopReason,
    toolCallCount: ctx.recordedToolCalls.length,
  });
  if (disposition === "cancelled") {
    return { result: createCancelledResult(), stoppedForClientTool: false };
  }
  return {
    result: completedPromptResult(settled.turn, ctx.recordedToolCalls),
    stoppedForClientTool: false,
  };
}

function resultFromCaughtTurn(ctx: TurnContext, intentionalDelegatedStop: boolean): AgentResult {
  const status = caughtTurnOutcome({
    userCancelled: ctx.isCancelRequested(),
    intentionalDelegatedStop,
    toolChoice: ctx.input.toolChoice,
    recordedToolNames: recordedNames(ctx),
  });
  if (status === "completed") {
    return createCompletedResult({ text: "", toolCalls: ctx.recordedToolCalls });
  }
  if (status === "cancelled") return createCancelledResult();
  return createFailedResult();
}

function completedPromptResult(
  turn: AgentTurnResult,
  recordedToolCalls: readonly RecordedToolCall[],
): AgentResult {
  return createCompletedResult(
    {
      text: turn.agentText,
      thoughtText: turn.thoughtText.length > 0 ? turn.thoughtText : undefined,
      toolCalls: recordedToolCalls.length > 0 ? recordedToolCalls : undefined,
    },
    usageFromPrompt(turn),
  );
}

function usageFromPrompt(turn: AgentTurnResult): AgentExecutionUsage | undefined {
  const usage = turn.prompt.usage;
  if (usage === undefined) return undefined;
  return {
    inputTokens: usage.inputTokens,
    outputTokens: usage.outputTokens,
    totalTokens: usage.totalTokens,
    thoughtTokens: usage.thoughtTokens,
  };
}

function finishDelegatedClientTurn(ctx: TurnContext, result: AgentResult): AgentResult {
  ctx.queue.push(createTerminalEvent(result, ctx.runId));
  ctx.queue.close();
  ctx.input.signal?.removeEventListener("abort", ctx.abortListener);
  ctx.onSessionReleased();
  // Microtasks flush response.completed first. SIGINT in this turn closes the SSE early.
  setImmediate(() => {
    try {
      ctx.client.cancelTurn(ctx.acpSessionId);
    } catch {
      // Codex already has the tool call. Process cleanup must not reopen the turn.
    }
    releaseAcpProcess(ctx.client, ctx.bridge);
  });
  return result;
}

async function finishPromptTurn(ctx: TurnContext, result: AgentResult): Promise<AgentResult> {
  const finished = await releasePromptProcess(ctx, result);
  ctx.queue.push(createTerminalEvent(finished.result, ctx.runId));
  ctx.queue.close();
  if (finished.cleanupFailure !== undefined) throw cleanupError(finished.cleanupFailure);
  return finished.result;
}

async function releasePromptProcess(
  ctx: TurnContext,
  result: AgentResult,
): Promise<{ result: AgentResult; cleanupFailure: unknown }> {
  let cleanupFailure: unknown;
  let settled = result;
  try {
    if (ctx.bridge) await ctx.bridge.dispose().catch(() => {});
    await ctx.client.terminate();
  } catch (error) {
    cleanupFailure = error;
    settled = createFailedResult();
  } finally {
    ctx.input.signal?.removeEventListener("abort", ctx.abortListener);
    ctx.onSessionReleased();
  }
  return { result: settled, cleanupFailure };
}

function releaseAcpProcess(
  client: AntigravityAcpClient,
  bridge: AntigravityMcpBridge | undefined,
): void {
  void (async () => {
    if (bridge) await bridge.dispose().catch(() => {});
    await client.terminate().catch(() => {});
  })();
}

function cleanupError(error: unknown): Error {
  return error instanceof Error
    ? error
    : new Error("ACP process cleanup failed.", { cause: error });
}

function recordedNames(ctx: TurnContext): string[] {
  return ctx.recordedToolCalls.map((call) => call.name);
}
