import { describe, expect, it } from "vitest";
import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import {
  enforceClientToolContract,
  validateClientToolCall,
} from "../../src/client-mcp/client-tool-contract.js";

const codexTools: readonly CursorClientToolSpec[] = [
  { name: "exec", inputFormat: "grammar" },
  {
    name: "apply_patch",
    inputFormat: "grammar",
  },
  {
    name: "client_shell",
    parameters: {
      type: "object",
      properties: { command: { type: "string" } },
      required: ["command"],
    },
  },
];

describe("client-tool-contract", () => {
  it("passes valid exec and apply_patch payloads unchanged", () => {
    const execCall = enforceClientToolContract(
      {
        name: "exec",
        arguments: {
          input:
            'const result = await tools.exec_command({cmd:"git status -sb"}); text(result);',
        },
      },
      codexTools,
    );
    expect(execCall.rawInput).toContain("tools.exec_command");
    expect(validateClientToolCall(execCall, codexTools)).toBeNull();

    const patchCall = enforceClientToolContract(
      {
        name: "apply_patch",
        arguments: {
          input: "*** Begin Patch\n*** End Patch\n",
        },
      },
      codexTools,
    );
    expect(patchCall.rawInput).toContain("*** Begin Patch");
    expect(validateClientToolCall(patchCall, codexTools)).toBeNull();
  });

  it("replaces invalid exec with text() and drops the forbidden script", () => {
    const forbidden =
      'await tools.client_write({path:"x",fileText:"y"}); text("ok");';
    const enforced = enforceClientToolContract(
      { name: "exec", arguments: { input: forbidden } },
      codexTools,
    );
    expect(enforced.name).toBe("exec");
    expect(enforced.rawInput).not.toContain("await tools.client_write");
    expect(enforced.rawInput).not.toContain("fileText");
    expect(enforced.rawInput).toMatch(/^text\(/);
    expect(enforced.rawInput).toContain("CONTRACT VIOLATION");
  });

  it("replaces raw shell in exec with a contract correction", () => {
    const enforced = enforceClientToolContract(
      { name: "exec", arguments: { input: "git status -sb" } },
      codexTools,
    );
    expect(enforced.rawInput).not.toBe("git status -sb");
    expect(enforced.rawInput).not.toMatch(/^git status/);
    expect(enforced.rawInput).toContain("tools.exec_command");
  });
});
