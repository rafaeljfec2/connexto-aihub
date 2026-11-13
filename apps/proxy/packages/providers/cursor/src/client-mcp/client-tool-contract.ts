import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import type { CursorToolCall } from "@agent-runtime-gateway/compatibility-openai";
import { isRecord } from "@agent-runtime-gateway/compatibility-openai";
import { attachGrammarInput, grammarSourceFromArguments } from "./grammar-input.js";
import { resolveInventoryToolName } from "./normalize.js";

export interface ClientToolContractViolation {
  readonly rule: string;
  readonly requiredFormat: string;
}

const FORBIDDEN_EXEC_SUBSTRINGS = [
  "tools.client_write",
  "tools.client",
  "tools.read",
  "node:fs",
  "readFileSync",
  "require(",
  "import(",
] as const;

const PATCH_MARKER = "*** Begin Patch";

export function mandatoryContractPromptBlock(
  clientTools: readonly CursorClientToolSpec[],
): string {
  const exactTools = clientTools
    .map((tool) => tool.name.trim())
    .filter((name) => name.length > 0)
    .join(", ");
  return [
    "GPT-6 OUTPUT CONTRACT (MANDATORY).",
    "This block is law, not advice. The proxy validates every mcp payload before Codex runs it.",
    "One tool per turn. toolName must be from the client inventory only.",
    "Assistant must include a brief natural-language update before mcp (for the user); only the mcp payload is validated here.",
    "Do not narrate SDK usage, editor tools, or Cursor-native names inside the mcp payload.",
    `Inventory: ${exactTools || "(none)"}.`,
    "function tools: args is a JSON object matching the schema. No JavaScript modules inside args.",
    "exec: args.input is Codex code-mode JavaScript. Shell uses await tools.exec_command({cmd:\"...\"}); then text(result).",
    `Forbidden inside exec: ${FORBIDDEN_EXEC_SUBSTRINGS.join(", ")}.`,
    `apply_patch: args.input must include ${PATCH_MARKER}, or call await tools.apply_patch(patchText) from exec.`,
    "Violations are discarded. The proxy forwards a replacement exec whose only output is text(\"...\") describing the broken rule.",
  ].join("\n");
}

export function validateClientToolCall(
  call: CursorToolCall,
  clientTools: readonly CursorClientToolSpec[],
): ClientToolContractViolation | null {
  const resolvedName = resolveInventoryToolName(call.name, clientTools);
  if (!resolvedName) {
    return {
      rule: "toolName is not in the client inventory.",
      requiredFormat: 'mcp with toolName from the inventory and providerIdentifier "client".',
    };
  }
  const spec = clientTools.find((tool) => tool.name === resolvedName);
  if (!spec) return null;

  if (spec.inputFormat === "grammar" || resolvedName === "apply_patch") {
    return validateGrammarToolCall(call, resolvedName, clientTools);
  }
  return validateFunctionToolCall(call);
}

export function enforceClientToolContract(
  call: CursorToolCall,
  clientTools: readonly CursorClientToolSpec[],
): CursorToolCall {
  const withGrammar = attachGrammarInput(call, clientTools);
  const violation = validateClientToolCall(withGrammar, clientTools);
  if (!violation) return withGrammar;
  return correctionExecCall(withGrammar, violation, clientTools);
}

function validateFunctionToolCall(call: CursorToolCall): ClientToolContractViolation | null {
  if (!isRecord(call.arguments)) {
    return {
      rule: "function tool args must be a JSON object.",
      requiredFormat: '{"providerIdentifier":"client","toolName":"<name>","args":{...schema fields...}}',
    };
  }
  const embedded = findForbiddenSubstring(JSON.stringify(call.arguments));
  if (embedded) {
    return {
      rule: `function tool args must not embed Codex exec scripts (found ${embedded}).`,
      requiredFormat: "Use toolName exec with args.input JavaScript, or a function tool with plain JSON fields.",
    };
  }
  if (call.name === "client_shell") {
    const command = firstString(call.arguments, "command");
    if (!command) {
      return {
        rule: "client_shell args.command is required.",
        requiredFormat: '{"command":"git status -sb"}',
      };
    }
    if (looksLikeCodexExecScript(command)) {
      return {
        rule: "client_shell must not receive Codex exec JavaScript.",
        requiredFormat: 'Use toolName exec with const result = await tools.exec_command({cmd:"..."}); text(result);',
      };
    }
  }
  return null;
}

function validateGrammarToolCall(
  call: CursorToolCall,
  resolvedName: string,
  clientTools: readonly CursorClientToolSpec[],
): ClientToolContractViolation | null {
  const source = grammarSource(call);
  if (!source.trim()) {
    return {
      rule: `${resolvedName} requires non-empty custom_tool_call input.`,
      requiredFormat:
        resolvedName === "apply_patch"
          ? `${PATCH_MARKER}\\n*** Update File: path\\n...\\n*** End Patch`
          : '{ "input": "<raw source>" }',
    };
  }

  if (resolvedName === "apply_patch") {
    return validateApplyPatchSource(source, clientTools);
  }
  if (resolvedName === "exec") {
    return validateExecSource(source, clientTools);
  }
  return null;
}

function validateApplyPatchSource(
  source: string,
  clientTools: readonly CursorClientToolSpec[],
): ClientToolContractViolation | null {
  if (source.includes(PATCH_MARKER)) return null;
  if (source.includes("tools.apply_patch")) return null;
  const hasExec = clientTools.some(
    (tool) => tool.name.trim() === "exec" && tool.inputFormat === "grammar",
  );
  return {
    rule: "apply_patch input must be a patch envelope or tools.apply_patch(...) inside exec.",
    requiredFormat: hasExec
      ? `${PATCH_MARKER} ... *** End Patch, or exec with await tools.apply_patch(patchText).`
      : `${PATCH_MARKER} ... *** End Patch.`,
  };
}

function validateExecSource(
  source: string,
  clientTools: readonly CursorClientToolSpec[],
): ClientToolContractViolation | null {
  const forbidden = findForbiddenSubstring(source);
  if (forbidden) {
    return {
      rule: `exec must not use ${forbidden}.`,
      requiredFormat:
        'const result = await tools.exec_command({cmd:"<shell command>"}); text(result);',
    };
  }
  if (looksLikeRawShell(source) && !source.includes("tools.exec_command")) {
    return {
      rule: "exec must not pass raw shell command lines.",
      requiredFormat:
        'const result = await tools.exec_command({cmd:"<shell command>"}); text(result);',
    };
  }
  if (sourceIncludesFileWriteWithoutPatch(source) && !source.includes("tools.apply_patch")) {
    const patchTool = clientTools.find((tool) => tool.name.trim() === "apply_patch");
    if (patchTool) {
      return {
        rule: "exec must not invent file writes; use apply_patch or a declared file tool.",
        requiredFormat: `${PATCH_MARKER} via toolName apply_patch, or await tools.apply_patch(patchText) inside exec.`,
      };
    }
  }
  return null;
}

function correctionExecCall(
  original: CursorToolCall,
  violation: ClientToolContractViolation,
  clientTools: readonly CursorClientToolSpec[],
): CursorToolCall {
  const execName =
    clientTools.find((tool) => tool.name.trim() === "exec")?.name ?? "exec";
  const message = `CONTRACT VIOLATION: ${violation.rule} Required format: ${violation.requiredFormat}`;
  const script = `text(${JSON.stringify(message)});`;
  const corrected: CursorToolCall = {
    ...original,
    name: execName,
    arguments: { input: script },
    rawInput: script,
  };
  return attachGrammarInput(corrected, clientTools);
}

function grammarSource(call: CursorToolCall): string {
  if (typeof call.rawInput === "string" && call.rawInput.trim()) {
    return call.rawInput;
  }
  return grammarSourceFromArguments(call.arguments);
}

function findForbiddenSubstring(source: string): string | null {
  for (const token of FORBIDDEN_EXEC_SUBSTRINGS) {
    if (source.includes(token)) return token;
  }
  return null;
}

function looksLikeRawShell(source: string): boolean {
  const trimmed = source.trim();
  if (/^(git|bash|sh|cd|npm|pnpm|yarn|node|python3?)\b/.test(trimmed)) return true;
  if (/(^|\n)\s*(git|curl|wget|sudo)\s+/.test(source)) return true;
  if (/\|\|/.test(source) || /&&/.test(source)) return true;
  if (/^cat\s+</.test(trimmed)) return true;
  return false;
}

function looksLikeCodexExecScript(source: string): boolean {
  return (
    source.includes("tools.exec_command") ||
    source.includes("text(") ||
    source.includes("await tools.")
  );
}

function sourceIncludesFileWriteWithoutPatch(source: string): boolean {
  return /\b(writeFile|createWriteStream|client_write|fs\.write)\b/.test(source);
}

function firstString(args: Record<string, unknown>, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = args[key];
    if (typeof value === "string" && value.trim()) return value.trim();
  }
  return undefined;
}
