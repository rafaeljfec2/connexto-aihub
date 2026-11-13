import { performance } from "node:perf_hooks";
import type { Response } from "express";
import type { AgentRun } from "@agent-runtime-gateway/core";
import {
  enforceStructuredOutputFormat,
  openAiErrorPayload,
  responseObject,
} from "@agent-runtime-gateway/compatibility-openai";
import { collectRunOutput } from "../../utils/run.js";
import { getLogger } from "../../utils/logger.js";
import { initSseHeaders } from "../../utils/sse.js";
import type { ResponseDeliveryContext } from "./delivery-context.js";

export type { ResponseDeliveryContext } from "./delivery-context.js";
export {
  extractActiveResponseToolResults,
  readReasoningEffort,
  readToolChoice,
  toExecutorReasoningEffort,
} from "./delivery-context.js";
export { streamResponseRun } from "./stream-response-run.js";

/** Collects a non-streaming AgentRun, enforces structured output, and replies JSON. */
export async function deliverNonStreamingResponse(
  run: AgentRun,
  res: Response,
  ctx: ResponseDeliveryContext,
): Promise<void> {
  const runOutput = await collectRunOutput(run, res, ctx.signal);
  const providerDurationMs =
    Math.round((performance.now() - ctx.providerStart) * 100) / 100;

  if (runOutput === null) {
    if (ctx.signal.aborted) return;
    getLogger().warn(
      {
        requestId: ctx.requestId,
        provider: ctx.target.providerId,
        model: ctx.target.modelId,
        status: "failed",
        durationMs: providerDurationMs,
      },
      "provider execution failed",
    );
    return;
  }

  getLogger().info(
    {
      requestId: ctx.requestId,
      provider: ctx.target.providerId,
      model: ctx.target.modelId,
      status: "completed",
      durationMs: providerDurationMs,
    },
    "provider execution completed",
  );

  const responseText =
    runOutput.toolCalls.length === 0
      ? enforceStructuredOutputFormat(runOutput.text, ctx.textFormat)
      : runOutput.text;

  const resp = responseObject({
    id: ctx.id,
    created: ctx.created,
    model: ctx.prepared.model,
    text: responseText,
    toolCalls: runOutput.toolCalls,
    promptChars: ctx.prepared.promptChars,
    usage: runOutput.usage,
    reasoningEffort: ctx.reasoningEffort,
    metadata: ctx.prepared.responseMetadata,
  });

  if (ctx.prepared.storeResponse) {
    ctx.services.responseStore.set(ctx.id, {
      id: ctx.id,
      created: ctx.created,
      model: ctx.prepared.model,
      inputItems: ctx.prepared.responseInputItems ?? [],
      sdkSessionKey: ctx.sessionAffinity,
      ownerPrincipal: ctx.principal,
      responseData: resp,
    });
  }

  res.status(200).json(resp);
}

/** Forwards a Cursor upstream Responses payload without rewriting the client contract. */
export async function deliverPassthroughUpstreamResponse(
  upstream: globalThis.Response,
  res: Response,
  ctx: ResponseDeliveryContext,
): Promise<void> {
  const providerDurationMs =
    Math.round((performance.now() - ctx.providerStart) * 100) / 100;
  const contentType = upstream.headers.get("content-type") ?? "";

  if (contentType.includes("text/event-stream") && ctx.prepared.stream) {
    initSseHeaders(res);
    if (!upstream.body) {
      res.status(502).end();
      return;
    }
    for await (const chunk of upstream.body as AsyncIterable<Uint8Array>) {
      if (ctx.signal.aborted) break;
      res.write(chunk);
    }
    if (!ctx.signal.aborted) {
      res.end();
    }
    getLogger().info(
      {
        requestId: ctx.requestId,
        provider: ctx.target.providerId,
        model: ctx.target.modelId,
        status: "completed",
        durationMs: providerDurationMs,
        passthrough: true,
      },
      "provider stream completed",
    );
    return;
  }

  const rawText = await upstream.text();
  if (!upstream.ok) {
    res.status(upstream.status).type(contentType || "application/json").send(rawText);
    return;
  }

  let payload: Record<string, unknown>;
  try {
    payload = JSON.parse(rawText) as Record<string, unknown>;
  } catch {
    res.status(502).json(openAiErrorPayload("Invalid upstream response", "api_error"));
    return;
  }

  const stored = { ...payload, id: ctx.id, created: ctx.created, model: ctx.prepared.model };
  if (ctx.prepared.storeResponse) {
    ctx.services.responseStore.set(ctx.id, {
      id: ctx.id,
      created: ctx.created,
      model: ctx.prepared.model,
      inputItems: ctx.prepared.responseInputItems ?? [],
      sdkSessionKey: ctx.sessionAffinity,
      ownerPrincipal: ctx.principal,
      responseData: stored,
    });
  }

  getLogger().info(
    {
      requestId: ctx.requestId,
      provider: ctx.target.providerId,
      model: ctx.target.modelId,
      status: "completed",
      durationMs: providerDurationMs,
      passthrough: true,
    },
    "provider execution completed",
  );

  res.status(200).json(stored);
}
