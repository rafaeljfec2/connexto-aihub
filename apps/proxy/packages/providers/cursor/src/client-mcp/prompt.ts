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
    codexClientGuide(tools),
    openaiOutputContract(tools),
    "If LOCAL TOOL RESULT records are present, treat those tools as already executed by the outer client and continue from the result.",
    "",
    prompt,
    "",
    trailer,
  ].join("\n");
}

function mcpCallExample(tools: readonly CursorClientToolSpec[]): string {
  const functionTool = tools.find((tool) => tool.inputFormat !== "grammar");
  const tool = functionTool ?? tools[0];
  if (!tool) return '{"providerIdentifier":"client","toolName":"<name>","args":{}}';
  if (tool.inputFormat === "grammar") {
    return `{"providerIdentifier":"client","toolName":"${tool.name.trim()}","args":{"input":"<raw ${tool.name.trim()} source>"}}`;
  }
  return `{"providerIdentifier":"client","toolName":"${tool.name.trim()}","args":{...JSON object for function_call.arguments}}`;
}

function codexClientGuide(tools: readonly CursorClientToolSpec[]): string {
  const functionNames = tools
    .filter((tool) => tool.inputFormat !== "grammar")
    .map((tool) => tool.name.trim());
  const lines = [
    "CODEX CLIENT.",
    "Codex is the program on the user's machine. It executes the tool you select. You do not have its filesystem or its tool objects.",
    "One mcp call becomes one Codex tool call. Codex runs it, then the next request brings the result back.",
  ];
  if (tools.some((tool) => tool.name.trim() === "exec" && tool.inputFormat === "grammar")) {
    lines.push(
      "Codex exec: Codex evaluates args.input as JavaScript in a sandbox. The only helpers are tools.exec_command({cmd: string}) and text(value). exec_command runs a shell command and returns its output. text sends that output back to you. There is no filesystem API in that sandbox.",
      'Shell example: {"providerIdentifier":"client","toolName":"exec","args":{"input":"const result = await tools.exec_command({cmd:\\"git status -sb\\"}); text(result);"}}',
    );
  }
  if (functionNames.length > 0) {
    lines.push(
      `Codex function tools (${functionNames.join(", ")}): Codex runs them on the workspace. Put the tool's JSON arguments in mcp args. Do not reimplement them inside exec.`,
    );
  }
  return lines.join("\n");
}

function openaiOutputContract(tools: readonly CursorClientToolSpec[]): string {
  const lines = [
    "OPENAI OUTPUT CONTRACT.",
    'Transport is always SDK mcp with providerIdentifier "client".',
    "function tools: args is the JSON object stored as function_call.arguments. Match the tool schema. Do not nest it under another args key.",
    "custom tools (grammar): args is {\"input\":\"<source>\"}. That string is forwarded unchanged as custom_tool_call.input.",
    "Creating or editing files is a separate mcp call to a function tool. The exec runtime cannot write the workspace.",
  ];
  for (const tool of tools) {
    lines.push(toolContractLine(tool, tools));
  }
  return lines.join("\n");
}

function toolContractLine(
  tool: CursorClientToolSpec,
  tools: readonly CursorClientToolSpec[],
): string {
  const name = tool.name.trim();
  if (tool.inputFormat === "grammar") {
    return grammarContract(name, tools);
  }
  if (name === "client_shell") {
    return 'client_shell is a function_call. Example args: {"command":"git status -sb"}.';
  }
  const properties = schemaPropertyNames(tool.parameters);
  const propertyText =
    properties.length > 0 ? ` Properties: ${properties.join(", ")}.` : "";
  const description =
    tool.description && tool.description.trim().length > 0
      ? ` Codex description: ${tool.description.trim()}`
      : "";
  return `${name} is a function_call. Codex executes this tool on the workspace. args must satisfy its JSON schema.${propertyText}${description}`;
}

function grammarContract(
  name: string,
  tools: readonly CursorClientToolSpec[],
): string {
  if (name !== "exec") {
    return `${name} is a custom_tool_call. Put only the raw ${name} source in args.input.`;
  }
  const fileTools = tools
    .filter((tool) => tool.inputFormat !== "grammar")
    .map((tool) => tool.name.trim())
    .filter((toolName) => /write|edit|patch|strreplace/i.test(toolName));
  const fileRoute =
    fileTools.length > 0
      ? `Use ${fileTools.join(" or ")} for file changes.`
      : "Use a function tool from the inventory for file changes.";
  return [
    "exec is a custom_tool_call. args.input is JavaScript for the Codex exec runtime, not a shell command line and not a file writer.",
    'Example args.input: const result = await tools.exec_command({cmd:"git status -sb"}); text(result);',
    "Do not pass a raw bash pipeline, tools.client, tools.client_write, node:fs, import(), or apply_patch inside exec.",
    fileRoute,
  ].join(" ");
}

function schemaPropertyNames(parameters: unknown): string[] {
  if (!isRecord(parameters) || !isRecord(parameters.properties)) return [];
  return Object.keys(parameters.properties).slice(0, 12);
}
