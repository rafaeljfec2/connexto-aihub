import type { AgentResult } from "@agent-runtime-gateway/core";
import type { AntigravityAcpClient } from "./acp-client.js";

interface ArmedCancellation {
  readonly client: AntigravityAcpClient;
  readonly acpSessionId: string;
  readonly resultPromise: Promise<AgentResult>;
}

interface CancellationState {
  cancelRequested: boolean;
  cancellationFailure: unknown;
  cancellationPromise: Promise<void> | undefined;
  armed: ArmedCancellation | undefined;
  readonly cancelGraceMs: number;
}

export interface RunCancellation {
  isCancelRequested(): boolean;
  getCancellationFailure(): unknown;
  abortListener(): void;
  request(): Promise<void>;
  arm(input: ArmedCancellation & { readonly signal: AbortSignal | undefined }): void;
}

export function createRunCancellation(cancelGraceMs: number): RunCancellation {
  const state: CancellationState = {
    cancelRequested: false,
    cancellationFailure: undefined,
    cancellationPromise: undefined,
    armed: undefined,
    cancelGraceMs,
  };
  const request = (): Promise<void> => {
    state.cancelRequested = true;
    state.cancellationPromise ??= stopArmedTurn(state);
    return state.cancellationPromise;
  };
  const abortListener = (): void => {
    void request().catch((error: unknown) => {
      state.cancellationFailure = error;
    });
  };
  return {
    isCancelRequested: () => state.cancelRequested,
    getCancellationFailure: () => state.cancellationFailure,
    abortListener,
    request,
    arm(input) {
      state.armed = input;
      input.signal?.addEventListener("abort", abortListener, { once: true });
      if (input.signal?.aborted === true) abortListener();
    },
  };
}

async function stopArmedTurn(state: CancellationState): Promise<void> {
  const armed = state.armed;
  if (armed === undefined) return;
  await cancelOrTerminate(state, armed);
  const completed = await waitForCompletion(armed.resultPromise, state.cancelGraceMs);
  if (!completed) await terminateQuietly(state, armed.client);
  try {
    await armed.resultPromise;
  } catch (error) {
    state.cancellationFailure ??= error;
  }
}

async function cancelOrTerminate(
  state: CancellationState,
  armed: ArmedCancellation,
): Promise<void> {
  try {
    armed.client.cancelTurn(armed.acpSessionId);
  } catch (error) {
    state.cancellationFailure = error;
    try {
      await armed.client.terminate();
    } catch (cleanupError) {
      state.cancellationFailure = new AggregateError(
        [error, cleanupError],
        "ACP cancellation and process cleanup both failed.",
      );
    }
  }
}

async function terminateQuietly(
  state: CancellationState,
  client: AntigravityAcpClient,
): Promise<void> {
  try {
    await client.terminate();
  } catch (error) {
    state.cancellationFailure ??= error;
  }
}

async function waitForCompletion(result: Promise<AgentResult>, timeoutMs: number): Promise<boolean> {
  let timer: NodeJS.Timeout | undefined;
  const timeout = new Promise<boolean>((resolve) => {
    timer = setTimeout(() => resolve(false), timeoutMs);
  });
  const completed = await Promise.race([
    result.then(
      () => true,
      () => true,
    ),
    timeout,
  ]);
  if (timer !== undefined) clearTimeout(timer);
  return completed;
}
