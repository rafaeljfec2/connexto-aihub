import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";

export function edgeOwnedBridgePromptPrefix(
  prompt: string,
  clientTools: readonly CursorClientToolSpec[],
): string {
  if (clientTools.length === 0) return prompt;
  const exactNames = clientTools
    .map((tool) => tool.name.trim())
    .filter((name) => name.length > 0);
  const exactTools = exactNames.join(", ");
  const exampleTool = exactNames[0] ?? "exec";
  const grammarTool = clientTools.find((tool) => tool.inputFormat === "grammar");
  const callExample = grammarTool
    ? `{"providerIdentifier":"client","toolName":"${grammarTool.name}","args":{"input":"<raw ${grammarTool.name} source>"}}`
    : `{"providerIdentifier":"client","toolName":"${exampleTool}","args":{...matching the ${exampleTool} schema}}`;
  const trailer = [
    "LOCAL TOOL REQUIRED.",
    "The only callable tool in this runtime is named mcp.",
    "You do not have GetDynamicTools, CallDynamicTool, GetMcpTools, or CallMcpTool. Those names fail and must not be used.",
    `Call mcp exactly once with ${callExample}.`,
    grammarTool
      ? `${grammarTool.name} is an OpenAI custom tool. Put its raw source in args.input. That string is forwarded unchanged as custom_tool_call.input.`
      : `toolName must be one of: ${exactTools}.`,
    "Do not answer with prose before that mcp call.",
  ].join("\n");
  return [
    "You are running through a local tool-enabled runtime behind an OpenAI-compatible client.",
    "The outer client owns local tool execution. The bridge must forward local operations; it must not execute SDK built-in shell/read/write/edit/glob/grep/ls/delete tools inside the bridge runtime.",
    `The outer client tools are: ${exactTools}.`,
    `Client-only tools are exposed through the client MCP server by exact name: ${exactTools}.`,
    "A local MCP server named client exposes forwarding tools such as client_shell, client_write, client_read, client_edit, client_delete, client_glob, client_grep, and the exact outer client tool names.",
    "Prefer exact client tools and dedicated client MCP tools such as write, read, edit, glob, grep, ls, delete, client_write, client_read, client_edit, client_glob, client_grep, client_ls, and client_delete before bash/client_shell. Use shell only for commands or when no dedicated client tool fits.",
    'Use SDK mcp with providerIdentifier "client" for every local operation. Do not use SDK built-in shell, write, edit, read, glob, grep, ls, delete, readLints, semSearch, todowrite, task, createPlan, generateImage, or recordScreen.',
    "If LOCAL TOOL RESULT records are present, treat those tools as already executed by the outer client and continue from the result.",
    "",
    prompt,
    "",
    trailer,
  ].join("\n");
}
