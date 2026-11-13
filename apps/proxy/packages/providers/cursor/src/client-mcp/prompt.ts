import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import { isRecord } from "@agent-runtime-gateway/compatibility-openai";

/** MCP transport complement for edge-owned client tools (not user task text). */
export function edgeOwnedBridgePromptPrefix(
  clientTools: readonly CursorClientToolSpec[],
): string {
  if (clientTools.length === 0) return "";
  const tools = clientTools.filter((tool) => tool.name.trim().length > 0);
  const exactTools = tools.map((tool) => tool.name.trim()).join(", ");
  const example = mcpCallShapeExample(tools);
  const functionExample = mcpCallFunctionExample(tools);
  const inventoryLines = tools.map((tool) => inventoryTransportLine(tool));
  const execGuide = execIsolateGuide(tools);

  return [
    "CLIENT MCP TRANSPORT (COMPLEMENTARY).",
    "Upstream Responses instructions remain authoritative; this block only describes how to reach the client runtime.",
    "Local tool execution belongs to the outer client; this runtime forwards one SDK mcp call per turn when needed.",
    userVisibleProgressBlock(),
    `Callable SDK tool name: mcp only. No GetDynamicTools, CallDynamicTool, GetMcpTools, or CallMcpTool.`,
    `Call mcp with ${example}. providerIdentifier must be "client". toolName must be one of: ${exactTools}.`,
    ...(functionExample ? [`Function tools use a separate top-level mcp call: ${functionExample}.`] : []),
    "function tools: mcp args is the JSON object stored as function_call.arguments (match the tool schema).",
    'custom tools (grammar): mcp args is {"input":"<raw source>"} forwarded as custom_tool_call.input (not JSON, not markdown fences).',
    ...inventoryLines,
    "The proxy converts your mcp payload into Responses function_call or custom_tool_call items; do not emit those item types yourself.",
    ...(execGuide ? [execGuide] : []),
    "If LOCAL TOOL RESULT records are present, treat those tools as already executed by the client and continue from the result.",
  ].join("\n");
}

function execIsolateGuide(tools: readonly CursorClientToolSpec[]): string | undefined {
  const hasExec = tools.some(
    (tool) => tool.name.trim() === "exec" && tool.inputFormat === "grammar",
  );
  if (!hasExec) return undefined;
  const nestedGrammarNames = tools
    .filter(
      (tool) =>
        tool.name.trim() !== "exec" &&
        (tool.inputFormat === "grammar" || tool.name.trim() === "apply_patch"),
    )
    .map((tool) => tool.name.trim())
    .filter((name) => name.length > 0);
  const functionOnlyNames = tools
    .filter((tool) => tool.inputFormat !== "grammar" && tool.name.trim() !== "apply_patch")
    .map((tool) => tool.name.trim())
    .filter((name) => name.length > 0);
  const nestedLine =
    nestedGrammarNames.length > 0
      ? `Inside exec, grammar tools only: ${nestedGrammarNames.join(", ")} via await tools.<name>(source) when declared in inventory.`
      : "";
  const functionLine =
    functionOnlyNames.length > 0
      ? `Not inside exec: ${functionOnlyNames.join(", ")} — use top-level mcp with toolName (function_call args), not tools.${functionOnlyNames[0]} in exec scripts.`
      : "";
  return [
    "EXEC ISOLATE (toolName exec): args.input is async JavaScript for the client V8 sandbox.",
    "Use await tools.exec_command({cmd, cwd}) for shell and filesystem inspection; emit results with text(value).",
    "Follow upstream Responses instructions for shell conventions (they override this block).",
    "Never use require, import(), node:fs, readFileSync, tools.read, tools.client, or undeclared tools.*.",
    nestedLine,
    functionLine,
    "The proxy rejects exec scripts with forbidden APIs before they reach the client.",
  ]
    .filter((line) => line.length > 0)
    .join(" ");
}

function userVisibleProgressBlock(): string {
  return [
    "Before each mcp call, write 1-3 short sentences in the user's language:",
    "what you understood, what you are doing now, and which inventory toolName comes next (plain language).",
    "Never run a client tool silently.",
  ].join(" ");
}

function mcpCallShapeExample(tools: readonly CursorClientToolSpec[]): string {
  const preferred =
    tools.find((tool) => tool.name.trim() === "exec" && tool.inputFormat === "grammar") ??
    tools.find((tool) => tool.inputFormat === "grammar") ??
    tools.find((tool) => tool.inputFormat !== "grammar") ??
    tools[0];
  if (!preferred) return '{"providerIdentifier":"client","toolName":"<name>","args":{}}';
  return mcpExampleForTool(preferred);
}

function mcpCallFunctionExample(tools: readonly CursorClientToolSpec[]): string | undefined {
  const functionTool = tools.find(
    (tool) =>
      tool.inputFormat !== "grammar" &&
      tool.name.trim() !== "exec" &&
      !(tool.name.trim() === "exec" && tool.inputFormat === "grammar"),
  );
  if (!functionTool) return undefined;
  const hasGrammarOrExec = tools.some(
    (tool) =>
      tool.name.trim() === "exec" ||
      tool.inputFormat === "grammar" ||
      tool.name.trim() === "apply_patch",
  );
  if (!hasGrammarOrExec) return undefined;
  return mcpExampleForTool(functionTool);
}

function mcpExampleForTool(tool: CursorClientToolSpec): string {
  const name = tool.name.trim();
  if (tool.inputFormat === "grammar" || name === "apply_patch") {
    return `{"providerIdentifier":"client","toolName":"${name}","args":{"input":"<raw ${name} source>"}}`;
  }
  return `{"providerIdentifier":"client","toolName":"${name}","args":{...schema fields}}`;
}

function inventoryTransportLine(tool: CursorClientToolSpec): string {
  const name = tool.name.trim();
  if (tool.inputFormat === "grammar") {
    if (name === "apply_patch") {
      return "apply_patch: custom_tool_call via args.input patch text (*** Begin Patch envelope).";
    }
    return `${name}: custom_tool_call via args.input raw source.`;
  }
  const properties = schemaPropertyNames(tool.parameters);
  const propertyText =
    properties.length > 0 ? ` Schema fields: ${properties.join(", ")}.` : "";
  return `${name}: function_call JSON args.${propertyText}`;
}

function schemaPropertyNames(parameters: unknown): string[] {
  if (!isRecord(parameters) || !isRecord(parameters.properties)) return [];
  return Object.keys(parameters.properties).slice(0, 12);
}

export function composeEdgeOwnedUserMessageText(params: {
  readonly prompt: string;
  readonly clientInstructions?: string;
  readonly clientTools: readonly CursorClientToolSpec[];
  readonly workingDirectory?: string;
}): string {
  const workspaceContext = params.workingDirectory
    ? `IDE workspace for client-side tools only: ${JSON.stringify(params.workingDirectory)}`
    : "";
  if (params.clientTools.length === 0) {
    return params.prompt;
  }
  return [
    params.clientInstructions,
    edgeOwnedBridgePromptPrefix(params.clientTools),
    params.prompt,
    workspaceContext.length > 0 ? workspaceContext : undefined,
  ]
    .filter((part): part is string => typeof part === "string" && part.length > 0)
    .join("\n\n");
}
