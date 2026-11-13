import { performance } from "node:perf_hooks";
import type { Response } from "express";
import type { AgentRun } from "@agent-runtime-gateway/core";
import { chatChunk, chatUsageChunk, type PreparedRequest } from "@agent-runtime-gateway/compatibility-openai";
import { openAiToolCallFromAgentEvent } from "../utils/agent-run-collect.js";
import { getLogger } from "../utils/logger.js";
import type { ProviderExecutionTarget } from "../utils/provider-target.js";
import {
  resolveStreamRunOutcome,
  startSsePing,
  writeSseDone,
} from "../utils/sse-stream.js";
import { initSseHeaders } from "../utils/sse.js";

export interface ChatCompletionStreamParams {
  readonly run: AgentRun;
  readonly res: Response;
  readonly signal: AbortSignal;
  readonly id: string;
  readonly created: number;
  readonly prepared: PreparedRequest;
  readonly providerStart: number;
  readonly requestId: string;
  readonly target: ProviderExecutionTarget;
}

export async function deliverChatCompletionStream(
  params: ChatCompletionStreamParams,
): Promise<void> {
  const { run, res, signal, id, created, prepared, providerStart, requestId, target } =
    params;

  initSseHeaders(res);
  res.write(chatChunk({ id, created, model: prepared.model, role: "assistant" }));

  let fullText = "";
  let firstDeltaMs: number | undefined;
  let toolCallIndex = 0;
  let streamFailed = false;
  const pingTimer = startSsePing(res);
  let terminalEmitted = false;

  const emitTerminal = (finishReason: "stop" | "tool_calls" | "length"): void => {
    if (terminalEmitted || res.writableEnded) return;
    terminalEmitted = true;
    res.write(
      chatChunk({
        id,
        created,
        model: prepared.model,
        finish: true,
        finishReason,
      }),
    );
    if (prepared.includeUsage) {
      res.write(
        chatUsageChunk({
          id,
          created,
          model: prepared.model,
          promptChars: prepared.promptChars,
          completionChars: fullText.length,
        }),
      );
    }
    writeSseDone(res);
  };

  try {
    for await (const event of run.events) {
      if (signal.aborted) break;
      if (event.type === "message.delta" && event.delta) {
        if (firstDeltaMs === undefined) {
          firstDeltaMs = Math.round((performance.now() - providerStart) * 100) / 100;
        }
        fullText += event.delta;
        res.write(chatChunk({ id, created, model: prepared.model, delta: event.delta }));
      } else if (event.type === "tool.call") {
        const toolCall = openAiToolCallFromAgentEvent(event);
        res.write(
          chatChunk({
            id,
            created,
            model: prepared.model,
            toolCall: { index: toolCallIndex, value: toolCall },
          }),
        );
        toolCallIndex += 1;
      } else if (event.type === "run.failed") {
        streamFailed = true;
        break;
      }
    }

    const outcome = await resolveStreamRunOutcome(run, signal, streamFailed);
    const providerDurationMs =
      Math.round((performance.now() - providerStart) * 100) / 100;

    getLogger().info(
      {
        requestId,
        provider: target.providerId,
        model: target.modelId,
        status: outcome.success ? "completed" : "failed",
        durationMs: providerDurationMs,
        timeToFirstDeltaMs: firstDeltaMs ?? providerDurationMs,
        streamFailed: outcome.streamFailed,
        aborted: outcome.aborted,
      },
      outcome.success ? "provider stream completed" : "provider stream ended incompletely",
    );

    const finishReason = outcome.success
      ? toolCallIndex > 0
        ? "tool_calls"
        : "stop"
      : "length";
    emitTerminal(finishReason);
  } catch (error) {
    getLogger().error(
      {
        requestId,
        provider: target.providerId,
        model: target.modelId,
        err: error,
      },
      "provider chat stream delivery failed",
    );
    emitTerminal("length");
  } finally {
    clearInterval(pingTimer);
    if (!terminalEmitted && !res.writableEnded) {
      emitTerminal("length");
    }
  }
}
