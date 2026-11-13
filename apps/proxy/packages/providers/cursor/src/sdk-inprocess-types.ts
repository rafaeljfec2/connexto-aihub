import type { AgentOptions } from "@cursor/sdk";
import type {
  AgentImageInput,
  CursorClientToolSpec,
} from "@agent-runtime-gateway/core";

export interface CursorSdkSendOptions {
  readonly mcpServers?: Record<string, unknown>;
}

export interface CursorSdkInProcessDependencies {
  createAgent(options: AgentOptions): Promise<{
    readonly agentId: string;
    send(
      message: unknown,
      options?: CursorSdkSendOptions,
    ): Promise<import("./sdk-inprocess-stream.js").CursorSdkRun>;
    close(): void;
  }>;
  deleteAgent(agentId: string): Promise<void>;
}

export interface CursorSdkInProcessInput {
  readonly apiKey: string;
  readonly model: string;
  readonly agentName: string;
  readonly prompt: string;
  readonly clientInstructions?: string;
  readonly workingDirectory?: string;
  readonly clientTools: readonly CursorClientToolSpec[];
  readonly images?: readonly AgentImageInput[];
  readonly signal?: AbortSignal;
}
