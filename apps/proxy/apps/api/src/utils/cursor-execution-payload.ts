import type { Request } from "express";
import type {
  CursorClientToolSpec,
  CursorExecutionContext,
} from "@agent-runtime-gateway/core";
import { isRecord, type PreparedRequest } from "@agent-runtime-gateway/compatibility-openai";
import {
  resolveCursorExecutionModeForRequest,
  type CursorExecutionMode,
} from "./cursor-execution-mode.js";

export function mapClientTools(
  tools: PreparedRequest["tools"],
): CursorClientToolSpec[] | undefined {
  if (!tools.length) return undefined;
  return tools.map((tool) => ({
    name: tool.name,
    ...(tool.description ? { description: tool.description } : {}),
    ...(tool.parameters !== undefined ? { parameters: tool.parameters } : {}),
  }));
}

function mapRawPassthroughTools(body: Record<string, unknown>): CursorClientToolSpec[] | undefined {
  const sources: unknown[] = [];
  if (Array.isArray(body.tools)) sources.push(...body.tools);
  if (Array.isArray(body.input)) {
    for (const item of body.input) {
      if (!isRecord(item) || item.type !== "additional_tools") continue;
      collectNestedToolNodes(item.tools, sources);
    }
  }
  if (sources.length === 0) return undefined;
  const specs: CursorClientToolSpec[] = [];
  const seen = new Set<string>();
  for (const tool of sources) {
    if (!isRecord(tool)) continue;
    if (tool.type === "namespace") continue;
    const name =
      typeof tool.name === "string" && tool.name.trim()
        ? tool.name.trim()
        : isRecord(tool.function) && typeof tool.function.name === "string"
          ? tool.function.name.trim()
          : "";
    if (!name || seen.has(name)) continue;
    seen.add(name);
    const format = isRecord(tool.format) ? tool.format : undefined;
    specs.push({
      name,
      ...(typeof tool.description === "string" ? { description: tool.description } : {}),
      ...(tool.parameters !== undefined
        ? { parameters: tool.parameters }
        : isRecord(tool.function) && tool.function.parameters !== undefined
          ? { parameters: tool.function.parameters }
          : {}),
      ...(format?.type === "grammar" ? { inputFormat: "grammar" as const } : {}),
    });
  }
  return specs.length > 0 ? specs : undefined;
}

function collectNestedToolNodes(value: unknown, out: unknown[]): void {
  if (!Array.isArray(value)) return;
  for (const node of value) {
    if (!isRecord(node)) continue;
    if (node.type === "namespace") {
      collectNestedToolNodes(node.tools, out);
      continue;
    }
    out.push(node);
  }
}

export function buildCursorExecutionContext(
  req: Request,
  prepared: PreparedRequest,
): CursorExecutionContext | undefined {
  const mode: CursorExecutionMode = resolveCursorExecutionModeForRequest(req);
  const clientTools =
    prepared.passthroughBody !== undefined
      ? (mapRawPassthroughTools(prepared.passthroughBody) ?? mapClientTools(prepared.tools))
      : mapClientTools(prepared.tools);
  return {
    mode,
    promptMode: prepared.prompt.mode ?? "ask",
    clientTools,
    // Edge-owned runs delegate tool execution to the hub client (Codex), not proxy disk.
    requiresLocalTool:
      mode === "edge_owned" ? false : prepared.requiresLocalTool,
  };
}
