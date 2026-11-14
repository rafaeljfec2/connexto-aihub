import type { AgentToolSpec } from "@agent-runtime-gateway/core";
import { sanitizeMcpToolSchema } from "./schema.js";

export function isCustomTool(spec: AgentToolSpec | undefined): boolean {
  return spec?.callKind === "custom" || spec?.inputFormat === "grammar";
}

/** MCP schema the ACP model fills. Grammar tools carry the raw source in `input`. */
export function mcpInputSchemaForTool(spec: AgentToolSpec): Record<string, unknown> {
  if (!isCustomTool(spec)) return sanitizeMcpToolSchema(spec.parameters);
  return {
    type: "object",
    properties: {
      input: {
        type: "string",
        description:
          'JavaScript source. Run a command with text(await tools.exec_command({cmd:"git status"})). run_command is not a tool.',
      },
    },
    required: ["input"],
  };
}

export function delegatedToolCallPayload(
  spec: AgentToolSpec | undefined,
  args: Record<string, unknown>,
): { arguments: string; callKind?: "custom" } {
  if (!isCustomTool(spec)) {
    return { arguments: JSON.stringify(args) };
  }
  const raw = typeof args.input === "string" ? args.input : JSON.stringify(args);
  return { arguments: raw, callKind: "custom" };
}
