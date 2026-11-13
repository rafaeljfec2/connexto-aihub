import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";

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
- Before each MCP call, provide 1-3 short progress sentences in the user's language.
- Never run a client tool silently.`;

/** MCP transport complement for edge-owned Codex client tools (not user task text). */
export function edgeOwnedBridgePromptPrefix(
  clientTools: readonly CursorClientToolSpec[],
): string {
  if (clientTools.length === 0) return "";
  const topLevel = topLevelToolNames(clientTools);
  const examples = mcpExampleLines(clientTools);
  return [
    CODEX_CODE_MODE_COMPLEMENT,
    topLevel.length > 0
      ? `Current top-level MCP toolName values: ${topLevel.join(", ")}.`
      : "No exec or wait in the current inventory; do not emit client tool MCP calls until they appear.",
    ...examples,
  ]
    .filter((line) => line.length > 0)
    .join("\n\n");
}

function topLevelToolNames(tools: readonly CursorClientToolSpec[]): string[] {
  const allowed = new Set(["exec", "wait"]);
  return tools
    .map((tool) => tool.name.trim())
    .filter((name) => name.length > 0 && allowed.has(name));
}

function mcpExampleLines(tools: readonly CursorClientToolSpec[]): string[] {
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
