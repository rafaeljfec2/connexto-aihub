import type { AgentToolChoice } from "@agent-runtime-gateway/core";
import { textBlock, type ContentBlock } from "./acp/index.js";

const MAX_PREAMBLE_CHARS = 400;

export const EXEC_FOLLOW_UP_TEXT = [
  "You stopped after a status sentence and did not call a tool.",
  "Call the MCP tool exec now. run_command and shell do not exist.",
  'Arguments are {"input":"<javascript>"} and the javascript must call tools.exec_command({cmd:"..."}) then text(result).',
  "Do not send another status sentence.",
].join(" ");

/** One retry when tools are available and the model only emitted a short preamble. */
export function shouldRetryExec(input: {
  readonly stoppedForClientTool: boolean;
  readonly status: string;
  readonly toolCallCount: number;
  readonly userCancelled: boolean;
  readonly toolCount: number;
  readonly toolChoice: AgentToolChoice | undefined;
  readonly text: string;
}): boolean {
  if (input.stoppedForClientTool || input.userCancelled) return false;
  if (input.status !== "completed" || input.toolCallCount > 0) return false;
  if (input.toolCount === 0 || input.toolChoice === "none") return false;
  const text = input.text.trim();
  return text.length > 0 && text.length <= MAX_PREAMBLE_CHARS;
}

export function execFollowUpBlocks(): readonly ContentBlock[] {
  return [textBlock(EXEC_FOLLOW_UP_TEXT)];
}
