import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import { isRecord } from "@agent-runtime-gateway/compatibility-openai";

export function edgeOwnedBridgePromptPrefix(
  prompt: string,
  clientTools: readonly CursorClientToolSpec[],
): string {
  if (clientTools.length === 0) return prompt;
  const tools = clientTools.filter((tool) => tool.name.trim().length > 0);
  const exactTools = tools.map((tool) => tool.name.trim()).join(", ");
  const example = mcpCallExample(tools);
  const trailer = [
    "LOCAL TOOL REQUIRED.",
    "The only callable tool in this runtime is named mcp.",
    "You do not have GetDynamicTools, CallDynamicTool, GetMcpTools, or CallMcpTool. Those names fail and must not be used.",
    `Call mcp exactly once with ${example}.`,
    `toolName must be one of: ${exactTools}.`,
    "Do not answer with prose before that mcp call.",
  ].join("\n");
  return [
    "You are running through an inference bridge behind an OpenAI Responses client.",
    "The outer client owns local tool execution. This runtime must not execute SDK built-in shell, read, write, edit, glob, grep, ls, or delete.",
    "Your job is to choose one outer-client tool and fill the OpenAI payload. The proxy turns that payload into a Responses function_call or custom_tool_call. Do not emit those item types yourself.",
    openaiOutputContract(tools),
    "If LOCAL TOOL RESULT records are present, treat those tools as already executed by the outer client and continue from the result.",
    "",
    prompt,
    "",
    trailer,
  ].join("\n");
}

function mcpCallExample(tools: readonly CursorClientToolSpec[]): string {
  const grammarTool = tools.find((tool) => tool.inputFormat === "grammar");
  const tool = grammarTool ?? tools[0];
  if (!tool) return '{"providerIdentifier":"client","toolName":"<name>","args":{}}';
  if (tool.inputFormat === "grammar") {
    return `{"providerIdentifier":"client","toolName":"${tool.name.trim()}","args":{"input":"<raw ${tool.name.trim()} source>"}}`;
  }
  return `{"providerIdentifier":"client","toolName":"${tool.name.trim()}","args":{...JSON object for function_call.arguments}}`;
}

function openaiOutputContract(tools: readonly CursorClientToolSpec[]): string {
  const lines = [
    "OPENAI OUTPUT CONTRACT.",
    'Transport is always SDK mcp with providerIdentifier "client".',
    "function tools: args is the JSON object stored as function_call.arguments. Match the tool schema. Do not nest it under another args key.",
    "custom tools (grammar): args is {\"input\":\"<source>\"}. That string is forwarded unchanged as custom_tool_call.input.",
  ];
  for (const tool of tools) {
    lines.push(toolContractLine(tool));
  }
  return lines.join("\n");
}

function toolContractLine(tool: CursorClientToolSpec): string {
  const name = tool.name.trim();
  if (tool.inputFormat === "grammar") {
    return grammarContract(name);
  }
  if (name === "client_shell") {
    return 'client_shell is a function_call. Example args: {"command":"git status -sb"}.';
  }
  const properties = schemaPropertyNames(tool.parameters);
  const propertyText =
    properties.length > 0 ? ` Properties: ${properties.join(", ")}.` : "";
  return `${name} is a function_call. args must satisfy its JSON schema.${propertyText}`;
}

function grammarContract(name: string): string {
  if (name !== "exec") {
    return `${name} is a custom_tool_call. Put only the raw ${name} source in args.input.`;
  }
  return [
    "exec is a custom_tool_call. args.input is JavaScript for the Codex exec runtime, not a shell command line.",
    'Example args.input: const result = await tools.exec_command({cmd:"git status -sb"}); text(result);',
    "Do not pass a raw bash pipeline, tools.client, or import() inside exec.",
  ].join(" ");
}

function schemaPropertyNames(parameters: unknown): string[] {
  if (!isRecord(parameters) || !isRecord(parameters.properties)) return [];
  return Object.keys(parameters.properties).slice(0, 12);
}
