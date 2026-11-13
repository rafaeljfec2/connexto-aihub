import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import type { CursorToolCall } from "@agent-runtime-gateway/compatibility-openai";
import { isRecord } from "@agent-runtime-gateway/compatibility-openai";

export function attachGrammarInput(
  call: CursorToolCall,
  clientTools: readonly CursorClientToolSpec[],
): CursorToolCall {
  const spec = clientTools.find((tool) => tool.name === call.name);
  if (spec?.inputFormat !== "grammar") return call;
  return { ...call, rawInput: grammarSourceFromArguments(call.arguments) };
}

export function grammarSourceFromArguments(args: Record<string, unknown>): string {
  const nested = nestedArguments(args);
  const source = firstString(nested, "input", "code", "source", "script");
  if (source) return source;
  return JSON.stringify(nested);
}

function nestedArguments(args: Record<string, unknown>): Record<string, unknown> {
  for (const key of ["args", "arguments"] as const) {
    const value = args[key];
    if (typeof value !== "string" || !value.trim()) continue;
    try {
      const parsed = JSON.parse(value) as unknown;
      if (isRecord(parsed)) return parsed;
    } catch {
      return args;
    }
  }
  return args;
}

function firstString(args: Record<string, unknown>, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = args[key];
    if (typeof value === "string" && value.trim()) return value.trim();
  }
  return undefined;
}
