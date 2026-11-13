import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import { isRecord } from "@agent-runtime-gateway/compatibility-openai";
import { mandatoryContractPromptBlock } from "./client-tool-contract.js";

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
    "When a client tool is needed: write the USER VISIBLE PROGRESS lines first, then call mcp exactly once.",
    `Call mcp with ${example}.`,
    `toolName must be one of: ${exactTools}.`,
  ].join("\n");
  return [
    "You are the Composer model emulating GPT-6 output for an OpenAI Responses client.",
    "Emit exactly one SDK mcp call per turn when a tool is needed, after USER VISIBLE PROGRESS. No Cursor editor or Agent SDK tool names.",
    userVisibleProgressBlock(),
    mandatoryContractPromptBlock(tools),
    "The outer client owns local tool execution. This runtime must not execute SDK built-in shell, read, write, edit, glob, grep, ls, or delete.",
    "The proxy turns your mcp payload into a Responses function_call or custom_tool_call. Do not emit those item types yourself.",
    codexClientGuide(tools),
    openaiOutputContract(tools),
    "If LOCAL TOOL RESULT records are present, treat those tools as already executed by the outer client and continue from the result.",
    "",
    prompt,
    "",
    trailer,
  ].join("\n");
}

function userVisibleProgressBlock(): string {
  return [
    "USER VISIBLE PROGRESS (REQUIRED).",
    "The user must always see what you are doing. Never run a client tool silently.",
    "Before each mcp call, write 1-3 short sentences in the same language as the user task:",
    "- what you understood from the request or the latest LOCAL TOOL RESULT;",
    "- what you are doing now;",
    "- which inventory toolName you will call next and why (plain language, not JSON).",
    "After LOCAL TOOL RESULT, briefly interpret the outcome and state the next step before another mcp (if needed).",
    "When no client tool is needed, end with a concise summary of the outcome.",
    "Progress text lives in assistant message text only. Never put progress, plans, or narration inside mcp args or exec scripts except text(result) outputs from Codex.",
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
  const lines = [
    "CODEX CLIENT INTERFACE.",
    "This block is the parser between you and Codex. Emit one mcp call whose payload matches the OpenAI Responses item Codex will run.",
    "OpenAI function tool: mcp args is the JSON object Codex stores as function_call.arguments.",
    "OpenAI custom tool: mcp args is {\"input\":\"<raw source>\"}. That string is custom_tool_call.input. It is not JSON and not a markdown fence.",
  ];
  if (tools.some((tool) => tool.name.trim() === "exec" && tool.inputFormat === "grammar")) {
    lines.push(
      "Codex code mode (openai/codex): the exec custom tool runs args.input as an async JavaScript module in a V8 isolate. No Node, no filesystem, no network, no console, no import().",
      "Nested Codex tools are methods on the global tools object. A freeform tool takes a string. A function tool takes an object.",
      "Shell: const result = await tools.exec_command({ cmd: \"git status -sb\" }); text(result);",
      "exec_command accepts cmd (required), and optionally workdir, shell, tty, yield_time_ms, and max_output_tokens.",
      "text(value) appends the output Codex returns to you. exit() stops the script.",
      "Do not call tools.client, tools.client_write, node:fs, or import(). Those are not Codex tools.",
    );
    const grammarNames = tools
      .filter((tool) => tool.inputFormat === "grammar" && tool.name.trim() !== "exec")
      .map((tool) => tool.name.trim());
    if (grammarNames.length > 0) {
      lines.push(
        `From inside exec, call freeform tools as await tools.${grammarNames[0]}(sourceText). You may also emit toolName "${grammarNames[0]}" with args.input set to that same raw source.`,
      );
    }
  }
  return lines.join("\n");
}

function openaiOutputContract(tools: readonly CursorClientToolSpec[]): string {
  const lines = [
    "OPENAI OUTPUT CONTRACT.",
    'Transport is always SDK mcp with providerIdentifier "client".',
    "function tools: args is the JSON object stored as function_call.arguments. Match the tool schema. Do not nest it under another args key.",
    "custom tools (grammar): args is {\"input\":\"<source>\"}. That string is forwarded unchanged as custom_tool_call.input.",
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
  if (name === "apply_patch") {
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
  if (name === "apply_patch") {
    return [
      "apply_patch is a custom_tool_call. args.input is the patch text, not JSON.",
      "Envelope: *** Begin Patch / file operations / *** End Patch.",
      "Operations: *** Add File: <relative path>, *** Delete File: <relative path>, *** Update File: <relative path>.",
      "Update hunks start with @@. Lines are context (space), deletion (-), or addition (+). Paths are relative.",
      "Inside exec the same text is the string argument: await tools.apply_patch(patchText).",
    ].join(" ");
  }
  if (name !== "exec") {
    return `${name} is a custom_tool_call. Put only the raw ${name} source in args.input.`;
  }
  const fileTools = tools
    .filter((tool) => tool.name.trim() !== "exec")
    .map((tool) => tool.name.trim())
    .filter((toolName) => /write|edit|patch|strreplace/i.test(toolName));
  const fileRoute =
    fileTools.length > 0
      ? `Use ${fileTools.join(" or ")} for file changes, either as its own mcp toolName or as await tools.${fileTools[0]}(...) inside exec.`
      : "Use a declared Codex tool for file changes.";
  return [
    "exec is a custom_tool_call. args.input is raw JavaScript for the Codex V8 isolate, not a shell command line.",
    'Example args.input: const result = await tools.exec_command({cmd:"git status -sb"}); text(result);',
    "Do not pass a raw bash pipeline, tools.client, tools.client_write, node:fs, or import() inside exec.",
    fileRoute,
  ].join(" ");
}

function schemaPropertyNames(parameters: unknown): string[] {
  if (!isRecord(parameters) || !isRecord(parameters.properties)) return [];
  return Object.keys(parameters.properties).slice(0, 12);
}
