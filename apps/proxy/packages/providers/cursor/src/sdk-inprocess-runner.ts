import type { CursorToolCall } from "@agent-runtime-gateway/compatibility-openai";
import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import type { CursorSdkCompletion } from "./sdk/types.js";
import {
  createIsolatedAgent,
  createSdkAgentResources,
  deleteIsolatedAgent,
} from "./sdk-inprocess-lifecycle.js";
import { streamSdkRun } from "./sdk-inprocess-stream.js";
import type {
  CursorSdkInProcessDependencies,
  CursorSdkInProcessInput,
  CursorSdkSendOptions,
} from "./sdk-inprocess-types.js";

export type { CursorSdkInProcessDependencies, CursorSdkInProcessInput, CursorSdkSendOptions };

const defaultDependencies: CursorSdkInProcessDependencies = {
  createAgent: createIsolatedAgent,
  deleteAgent: deleteIsolatedAgent,
};

interface ClientToolInventory {
  readonly clientTools: readonly CursorClientToolSpec[];
  tryCapture(call: CursorToolCall): boolean;
  peekCaptured(): CursorToolCall | null;
  waitForCapture(): Promise<CursorToolCall>;
}

export async function createCursorSdkInProcessCompletion(
  input: CursorSdkInProcessInput,
  dependencies: CursorSdkInProcessDependencies = defaultDependencies,
): Promise<CursorSdkCompletion> {
  input.signal?.throwIfAborted();
  const inventory = createClientToolInventory(input.clientTools);
  const resources = createSdkAgentResources(input, inventory, dependencies);

  try {
    const { agent, run } = await resources.start();
    return {
      agentId: agent.agentId,
      runId: run.id,
      stream: streamSdkRun(run, inventory, input, resources.cleanup),
      cancel: resources.cleanup,
    };
  } catch (error) {
    await resources.cleanup();
    throw error;
  }
}

function createClientToolInventory(
  clientTools: readonly CursorClientToolSpec[],
): ClientToolInventory {
  const names = new Set(
    clientTools.map((tool) => tool.name.trim()).filter((name) => name.length > 0),
  );
  let captured: CursorToolCall | null = null;
  let notifyCapture: ((call: CursorToolCall) => void) | undefined;
  const capturedPromise = new Promise<CursorToolCall>((resolve) => {
    notifyCapture = resolve;
  });
  return {
    clientTools,
    tryCapture(call: CursorToolCall): boolean {
      if (captured !== null || !names.has(call.name)) return false;
      captured = call;
      notifyCapture?.(call);
      return true;
    },
    peekCaptured(): CursorToolCall | null {
      return captured;
    },
    waitForCapture(): Promise<CursorToolCall> {
      return capturedPromise;
    },
  };
}
