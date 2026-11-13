import type { Response } from "express";
import { doneChunk } from "@agent-runtime-gateway/compatibility-openai";
import type { AgentRun } from "@agent-runtime-gateway/core";

export const SSE_PING_INTERVAL_MS = 10_000;
export const SSE_PING_FRAME = ": ping\n\n";

type FlushableResponse = Response & { flush?: () => void };

export function flushSse(res: Response): void {
  (res as FlushableResponse).flush?.();
}

export function startSsePing(res: Response): ReturnType<typeof setInterval> {
  return setInterval(() => {
    if (res.writableEnded) return;
    res.write(SSE_PING_FRAME);
    flushSse(res);
  }, SSE_PING_INTERVAL_MS);
}

export function writeSseDone(res: Response): void {
  if (res.writableEnded) return;
  res.write(doneChunk());
  res.end();
}

export interface StreamRunOutcome {
  readonly success: boolean;
  readonly aborted: boolean;
  readonly streamFailed: boolean;
}

export async function resolveStreamRunOutcome(
  run: AgentRun,
  signal: AbortSignal,
  streamFailed: boolean,
): Promise<StreamRunOutcome> {
  if (signal.aborted) {
    return { success: false, aborted: true, streamFailed };
  }
  if (streamFailed) {
    return { success: false, aborted: false, streamFailed: true };
  }
  const result = await run.wait();
  return {
    success: result.status === "completed",
    aborted: false,
    streamFailed: result.status !== "completed",
  };
}
