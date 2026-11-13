import { describe, expect, it } from "vitest";
import {
  composeEdgeOwnedUserMessageText,
  edgeOwnedBridgePromptPrefix,
} from "../../src/client-mcp/prompt.js";

const codexInventory = [
  { name: "exec", inputFormat: "grammar" as const },
  {
    name: "wait",
    parameters: {
      type: "object",
      properties: { cell_id: { type: "string" } },
      required: ["cell_id"],
    },
  },
];

describe("edgeOwnedBridgePromptPrefix", () => {
  it("describes Codex code mode without Cursor tool names or GPT-6 legacy blocks", () => {
    const complement = edgeOwnedBridgePromptPrefix(codexInventory);

    expect(complement).toContain("CODEX CODE MODE INTERFACE");
    expect(complement).toContain('providerIdentifier "client"');
    expect(complement).toContain("tools.exec_command");
    expect(complement).toContain("workdir");
    expect(complement).toContain("session_id");
    expect(complement).toContain("cell_id");
    expect(complement).toContain("text(value)");
    expect(complement).toContain("ALL_TOOLS");
    expect(complement).toContain("Never send toolName apply_patch through MCP");
    expect(complement).not.toContain("Composer model emulating GPT-6");
    expect(complement).not.toContain("git status -sb");
    expect(complement).not.toContain("GetDynamicTools");
    expect(complement).not.toMatch(/\bcwd\b/);
  });

  it("lists only exec and wait as top-level MCP toolNames", () => {
    const complement = edgeOwnedBridgePromptPrefix([
      ...codexInventory,
      {
        name: "client_shell",
        parameters: {
          type: "object",
          properties: { command: { type: "string" } },
          required: ["command"],
        },
      },
    ]);

    expect(complement).toContain("Current top-level MCP toolName values: exec, wait.");
    expect(complement).toMatch(/"toolName":"exec"/);
    expect(complement).toMatch(/"toolName":"wait"/);
    expect(complement).not.toContain('"toolName":"client_shell"');
    expect(complement).not.toContain("client_shell — use top-level mcp");
  });

  it("states wait versus write_stdin roles", () => {
    const complement = edgeOwnedBridgePromptPrefix(codexInventory);
    expect(complement).toContain("They are not interchangeable");
    expect(complement).toContain("write_stdin interacts with shell processes");
  });
});

describe("composeEdgeOwnedUserMessageText", () => {
  it("keeps task text alone when there are no client tools", () => {
    expect(
      composeEdgeOwnedUserMessageText({
        prompt: "Only the user turn",
        clientTools: [],
        clientInstructions: "Always prefix shell commands with rtk",
      }),
    ).toBe("Only the user turn");
  });

  it("orders instructions, complement, and input when client tools are active", () => {
    const text = composeEdgeOwnedUserMessageText({
      prompt: "apply a one-line fix",
      clientInstructions: "Always prefix shell commands with rtk\nYou are Codex.",
      clientTools: codexInventory,
      workingDirectory: "/workspace/repo",
    });

    const rtkAt = text.indexOf("Always prefix shell commands with rtk");
    const complementAt = text.indexOf("CODEX CODE MODE INTERFACE");
    const inputAt = text.indexOf("apply a one-line fix");
    const workdirAt = text.indexOf("IDE workspace");

    expect(rtkAt).toBeGreaterThanOrEqual(0);
    expect(complementAt).toBeGreaterThan(rtkAt);
    expect(inputAt).toBeGreaterThan(complementAt);
    expect(workdirAt).toBeGreaterThan(inputAt);
  });

  it("omits instructions block when clientInstructions is missing", () => {
    const text = composeEdgeOwnedUserMessageText({
      prompt: "Ship it",
      clientTools: codexInventory,
    });
    expect(text.startsWith("CODEX CODE MODE INTERFACE")).toBe(true);
    expect(text).toContain("Ship it");
  });
});
