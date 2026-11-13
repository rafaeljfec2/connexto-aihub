import type { OpenAiToolSpec } from "../types.js";
import { isRecord } from "../common.js";
import { parseChatTools } from "./specs.js";

/**
 * Codex and similar clients often place the live tool inventory in
 * `input[]` items with `type: "additional_tools"` (optionally nested under
 * `type: "namespace"`), while leaving top-level `tools` empty.
 */
export function collectResponsesToolSpecs(body: Record<string, unknown>): OpenAiToolSpec[] {
  const fromTopLevel =
    body.tool_choice === "none" ? [] : parseChatTools(body.tools);
  const fromInput = flattenAdditionalToolsFromInput(body.input);
  if (fromTopLevel.length === 0) return fromInput;
  if (fromInput.length === 0) return fromTopLevel;
  return mergeToolSpecs(fromTopLevel, fromInput);
}

export function flattenAdditionalToolsFromInput(input: unknown): OpenAiToolSpec[] {
  if (!Array.isArray(input)) return [];
  const out: OpenAiToolSpec[] = [];
  for (const item of input) {
    if (!isRecord(item)) continue;
    if (item.type !== "additional_tools") continue;
    collectToolNodes(item.tools, out);
  }
  return out;
}

function collectToolNodes(value: unknown, out: OpenAiToolSpec[]): void {
  if (!Array.isArray(value)) return;
  for (const node of value) {
    if (!isRecord(node)) continue;
    const type = typeof node.type === "string" ? node.type.trim() : "";
    if (type === "namespace") {
      collectToolNodes(node.tools, out);
      continue;
    }
    const spec = toolNodeToSpec(node);
    if (spec) out.push(spec);
  }
}

function toolNodeToSpec(node: Record<string, unknown>): OpenAiToolSpec | undefined {
  const fn = isRecord(node.function) ? node.function : node;
  const name =
    typeof fn.name === "string" && fn.name.trim()
      ? fn.name.trim()
      : typeof node.name === "string" && node.name.trim()
        ? node.name.trim()
        : "";
  if (!name) return undefined;
  const description =
    typeof fn.description === "string"
      ? fn.description
      : typeof node.description === "string"
        ? node.description
        : undefined;
  const parameters =
    fn.parameters !== undefined
      ? fn.parameters
      : node.parameters !== undefined
        ? node.parameters
        : undefined;
  return {
    name,
    ...(description ? { description } : {}),
    ...(parameters !== undefined ? { parameters } : {}),
  };
}

function mergeToolSpecs(
  primary: OpenAiToolSpec[],
  secondary: OpenAiToolSpec[],
): OpenAiToolSpec[] {
  const seen = new Set(primary.map((tool) => tool.name));
  const merged = [...primary];
  for (const tool of secondary) {
    if (seen.has(tool.name)) continue;
    seen.add(tool.name);
    merged.push(tool);
  }
  return merged;
}
