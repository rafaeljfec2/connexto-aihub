import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import { isRecord } from "@agent-runtime-gateway/compatibility-openai";

const CODEX_CODE_MODE_COMPLEMENT = `CODEX CODE MODE INTERFACE (COMPLEMENTARY).

Upstream Codex instructions remain authoritative. This block defines only the MCP transport required to access the Codex runtime.

TRANSPORT:
- Invoke only the SDK mcp tool with providerIdentifier "client".
- Emit exactly one MCP call per turn when a tool is needed.
- Use only top-level Codex tools exec and wait, as declared in the current inventory.
- exec is custom/freeform: args is {"input":"<raw JavaScript source>"}.
- wait is a function tool: args must match its declared JSON schema.
- The proxy converts MCP calls into Responses function_call or custom_tool_call. Never emit these items directly.
- Never invoke nested tools directly through MCP.
- Never invoke Cursor-native tools (read, edit, shell, grep, or other non-mcp SDK tools).

EXEC:
- Use exec/wait to run all other tools; do not attempt to use other tools directly.
- Execute raw JavaScript in the Codex V8 isolate, not Node.js.
- Nested tools are methods on the global tools object.
- Use only tools available in the nested tool catalog.
- Filter ALL_TOOLS by name and description when discovering deferred tools. Do not print the full catalog.
- Shell: use tools.exec_command with its declared schema, including cmd and optional workdir.
- Process interaction: use tools.write_stdin with its declared schema and the session_id returned by exec_command.
- Patching: inside exec, use tools.apply_patch with patch text when that nested tool is available. Never send toolName apply_patch through MCP.
- Follow the exact tool descriptions and schemas. Never invent tool names, parameters, or capabilities.

RUNTIME:
- JavaScript executes in a fresh asynchronous V8 isolate.
- No Node.js, filesystem, network, or console APIs.
- Do not use require, import(), node:fs, or undeclared APIs.
- Available global helpers include text, image, store, load, notify, exit, yield_control, setTimeout, clearTimeout, and ALL_TOOLS.
- Use text(value) to return results.

WAIT:
- Use wait only after exec reports a running cell ID.
- Pass the returned cell_id and arguments matching the declared wait schema.
- wait waits for more output from that JavaScript cell. It does not submit new JavaScript.
- write_stdin interacts with shell processes. They are not interchangeable.

RESULT HANDLING:
- Treat LOCAL TOOL RESULT as an operation already executed by the outer Codex client.
- Continue following the original Codex instructions.

USER VISIBILITY:
- Write in the user's language with a natural, collegial tone (Brazilian Portuguese when the user writes in PT-BR).
- Before each MCP call, add one brief human update: what you are doing now and why — not a workflow script.
- Do not repeat the same intent in back-to-back messages, cite skill or file paths, or open with robotic formulas ("Vou executar o fluxo...", "Iniciando conforme o skill...").
- On retries, say plainly what failed and what you are trying next; keep it to one short paragraph before the call.
- Never run a client tool silently.`;

/** MCP transport complement for edge-owned client tools (not user task text). */
export function edgeOwnedBridgePromptPrefix(
  clientTools: readonly CursorClientToolSpec[],
): string {
  if (clientTools.length === 0) return "";
  if (usesCodexCodeMode(clientTools)) {
    return codexBridgePromptSuffix(clientTools);
  }
  return legacyBridgePrompt(clientTools);
}

function usesCodexCodeMode(tools: readonly CursorClientToolSpec[]): boolean {
  return tools.some((tool) => {
    const name = tool.name.trim();
    return name === "exec" || name === "wait";
  });
}

function codexBridgePromptSuffix(clientTools: readonly CursorClientToolSpec[]): string {
  const topLevel = topLevelCodexToolNames(clientTools);
  const examples = codexMcpExampleLines(clientTools);
  return [
    CODEX_CODE_MODE_COMPLEMENT,
    `Current top-level MCP toolName values: ${topLevel.join(", ")}.`,
    ...examples,
  ]
    .filter((line) => line.length > 0)
    .join("\n\n");
}

function legacyBridgePrompt(clientTools: readonly CursorClientToolSpec[]): string {
  const tools = clientTools.filter((tool) => tool.name.trim().length > 0);
  const exactTools = tools.map((tool) => tool.name.trim()).join(", ");
  const example = legacyMcpCallShapeExample(tools);
  const inventoryLines = tools.map((tool) => legacyInventoryTransportLine(tool));
  return [
    "CLIENT MCP TRANSPORT (COMPLEMENTARY).",
    "Upstream Responses instructions remain authoritative; this block only describes how to reach the client runtime.",
    "Before each mcp call, write one brief natural update in the user's language (no duplicate narration or skill-path quotes); never run a client tool silently.",
    `Callable SDK tool name: mcp only. providerIdentifier must be "client".`,
    `Call mcp with ${example}. toolName must be one of: ${exactTools}.`,
    "function tools: mcp args is the JSON object stored as function_call.arguments (match the tool schema).",
    'custom tools (grammar): mcp args is {"input":"<raw source>"} forwarded as custom_tool_call.input.',
    ...inventoryLines,
    "The proxy converts your mcp payload into Responses function_call or custom_tool_call items; do not emit those item types yourself.",
    "If LOCAL TOOL RESULT records are present, treat those tools as already executed by the client and continue from the result.",
  ]
    .filter((line) => line.length > 0)
    .join("\n");
}

function topLevelCodexToolNames(tools: readonly CursorClientToolSpec[]): string[] {
  const allowed = new Set(["exec", "wait"]);
  return tools
    .map((tool) => tool.name.trim())
    .filter((name) => name.length > 0 && allowed.has(name));
}

function codexMcpExampleLines(tools: readonly CursorClientToolSpec[]): string[] {
  const lines: string[] = [];
  if (tools.some((tool) => tool.name.trim() === "exec")) {
    lines.push(
      'Example exec MCP args shape: {"providerIdentifier":"client","toolName":"exec","args":{"input":"<raw JavaScript source>"}}.',
    );
  }
  if (tools.some((tool) => tool.name.trim() === "wait")) {
    lines.push(
      'Example wait MCP args shape: {"providerIdentifier":"client","toolName":"wait","args":{...fields from wait schema, including cell_id}}.',
    );
  }
  return lines;
}

function legacyMcpCallShapeExample(tools: readonly CursorClientToolSpec[]): string {
  const preferred =
    tools.find((tool) => tool.inputFormat === "grammar") ??
    tools.find((tool) => tool.inputFormat !== "grammar") ??
    tools[0];
  if (!preferred) {
    return '{"providerIdentifier":"client","toolName":"<name>","args":{}}';
  }
  return legacyMcpExampleForTool(preferred);
}

function legacyMcpExampleForTool(tool: CursorClientToolSpec): string {
  const name = tool.name.trim();
  if (tool.inputFormat === "grammar" || name === "apply_patch") {
    return `{"providerIdentifier":"client","toolName":"${name}","args":{"input":"<raw ${name} source>"}}`;
  }
  return `{"providerIdentifier":"client","toolName":"${name}","args":{...schema fields}}`;
}

function legacyInventoryTransportLine(tool: CursorClientToolSpec): string {
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
