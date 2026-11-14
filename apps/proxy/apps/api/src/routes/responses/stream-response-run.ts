import { performance } from "node:perf_hooks";
import type { Response } from "express";
import type { AgentRun } from "@agent-runtime-gateway/core";
import {
  doneChunk,
  responseCreatedEvents,
  responseDeltaEvent,
  responseDoneEvents,
  responseObject,
  responseTextStartEvents,
  responseToolCallEvents,
} from "@agent-runtime-gateway/compatibility-openai";
import { openAiToolCallFromAgentEvent } from "../../utils/agent-run-collect.js";
import { getLogger } from "../../utils/logger.js";
import { flushSse, startSsePing, writeSseFrame } from "../../utils/sse-stream.js";
import { initSseHeaders } from "../../utils/sse.js";
import type { ResponseDeliveryContext } from "./delivery-context.js";

function writeStreamTerminal(
  res: Response,
  ctx: ResponseDeliveryContext,
  fullText: string,
  toolCalls: ReturnType<typeof openAiToolCallFromAgentEvent>[],
  terminal: {
    status: "completed" | "incomplete";
    incompleteDetails?: { reason: string };
  },
): void {
  for (const frame of responseDoneEvents({
    id: ctx.id,
    created: ctx.created,
    model: ctx.prepared.model,
    text: fullText,
    toolCalls,
    promptChars: ctx.prepared.promptChars,
    metadata: ctx.prepared.responseMetadata,
    textStarted: true,
    textOutputIndex: 0,
    status: terminal.status,
    incompleteDetails: terminal.incompleteDetails ?? null,
  })) {
    writeSseFrame(res, frame);
  }
  writeSseFrame(res, doneChunk());
  flushSse(res);
  res.end();
}

/** Streams an executed AgentRun as OpenAI Responses SSE frames and stores if requested. */
export async function streamResponseRun(
  run: AgentRun,
  res: Response,
  ctx: ResponseDeliveryContext,
): Promise<void> {
  initSseHeaders(res);
  res.flushHeaders?.();

  for (const frame of responseCreatedEvents({
    id: ctx.id,
    created: ctx.created,
    model: ctx.prepared.model,
  })) {
    writeSseFrame(res, frame);
  }

  for (const frame of responseTextStartEvents({ id: ctx.id, outputIndex: 0 })) {
    writeSseFrame(res, frame);
  }

  let fullText = "";
  let firstDeltaMs: number | undefined;
  const toolCalls: ReturnType<typeof openAiToolCallFromAgentEvent>[] = [];
  let streamFailed = false;
  const pingTimer = startSsePing(res);
  let terminalEmitted = false;

  const emitTerminal = (
    terminalStatus: "completed" | "incomplete",
    incompleteReason?: string,
  ): void => {
    if (terminalEmitted || res.writableEnded) return;
    terminalEmitted = true;
    writeStreamTerminal(res, ctx, fullText, toolCalls, {
      status: terminalStatus,
      ...(incompleteReason ? { incompleteDetails: { reason: incompleteReason } } : {}),
    });
  };

  try {
    for await (const event of run.events) {
      if (ctx.signal.aborted) break;
      if (event.type === "message.delta" && event.delta) {
        if (firstDeltaMs === undefined) {
          firstDeltaMs = Math.round((performance.now() - ctx.providerStart) * 100) / 100;
        }
        fullText += event.delta;
        writeSseFrame(res, responseDeltaEvent({ id: ctx.id, delta: event.delta, outputIndex: 0 }));
      } else if (event.type === "tool.call") {
        const toolCall = openAiToolCallFromAgentEvent(event);
        toolCalls.push(toolCall);
        const outputIndex = toolCalls.length;
        for (const frame of responseToolCallEvents({
          id: ctx.id,
          toolCall,
          outputIndex,
        })) {
          writeSseFrame(res, frame);
        }
      } else if (event.type === "run.failed") {
        streamFailed = true;
        break;
      }
    }

    const providerDurationMs =
      Math.round((performance.now() - ctx.providerStart) * 100) / 100;

    let terminalStatus: "completed" | "incomplete" = "completed";
    let incompleteReason: string | undefined;
    if (ctx.signal.aborted) {
      terminalStatus = "incomplete";
      incompleteReason = "client_disconnected";
    } else if (streamFailed) {
      terminalStatus = "incomplete";
      incompleteReason = "provider_error";
    } else {
      const result = await run.wait();
      if (result.status !== "completed") {
        terminalStatus = "incomplete";
        incompleteReason = result.status === "cancelled" ? "cancelled" : "provider_error";
      }
    }

    const streamSucceeded = terminalStatus === "completed";
    getLogger().info(
      {
        requestId: ctx.requestId,
        provider: ctx.target.providerId,
        model: ctx.target.modelId,
        status: streamSucceeded ? "completed" : "failed",
        durationMs: providerDurationMs,
        timeToFirstDeltaMs: firstDeltaMs ?? providerDurationMs,
        incompleteReason,
      },
      streamSucceeded ? "provider stream completed" : "provider stream ended incompletely",
    );

    emitTerminal(terminalStatus, incompleteReason);

    if (streamSucceeded && ctx.prepared.storeResponse) {
      ctx.services.responseStore.set(ctx.id, {
        id: ctx.id,
        created: ctx.created,
        model: ctx.prepared.model,
        inputItems: ctx.prepared.responseInputItems ?? [],
        sdkSessionKey: ctx.sessionAffinity,
        ownerPrincipal: ctx.principal,
        responseData: responseObject({
          id: ctx.id,
          created: ctx.created,
          model: ctx.prepared.model,
          text: fullText,
          toolCalls,
          promptChars: ctx.prepared.promptChars,
          metadata: ctx.prepared.responseMetadata,
        }),
      });
    }
  } catch (error) {
    getLogger().error(
      {
        requestId: ctx.requestId,
        provider: ctx.target.providerId,
        model: ctx.target.modelId,
        err: error,
      },
      "provider stream delivery failed",
    );
    emitTerminal("incomplete", "provider_error");
  } finally {
    clearInterval(pingTimer);
    if (!terminalEmitted && !res.writableEnded) {
      emitTerminal("incomplete", "provider_error");
    }
  }
}
