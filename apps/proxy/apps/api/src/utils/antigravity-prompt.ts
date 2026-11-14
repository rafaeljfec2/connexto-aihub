import { instructionsToPlainText } from "@agent-runtime-gateway/compatibility-openai";

const ANTIGRAVITY_TOOL_LINE = [
  "To inspect the repo or run a command, call the MCP tool exec in this same turn.",
  "exec takes {\"input\":\"<javascript>\"}. That javascript must call tools.exec_command({cmd:\"...\"}) and text(result).",
  "run_command and shell are not available. A status sentence without exec does not finish the turn.",
  "Before exec, write one short sentence in the user's language and nothing else.",
].join("\n");

/** Hub passthrough prompt for Antigravity: instructions, then a tool hint, then the input. */
export function composeAntigravityUserMessage(params: {
  readonly passthrough: boolean;
  readonly instructions: unknown;
  /** True only when the ACP session will actually register connexto_tools. */
  readonly toolsActive: boolean;
  readonly inputText: string;
}): string {
  if (!params.passthrough) return params.inputText;
  const instructions = instructionsToPlainText(params.instructions).trim();
  const toolLine = params.toolsActive ? ANTIGRAVITY_TOOL_LINE : "";
  return [instructions, toolLine, params.inputText]
    .filter((part) => part.length > 0)
    .join("\n\n");
}
