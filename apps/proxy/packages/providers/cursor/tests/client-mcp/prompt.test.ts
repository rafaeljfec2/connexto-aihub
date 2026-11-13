import { describe, expect, it } from "vitest";
import {
  composeEdgeOwnedUserMessageText,
  edgeOwnedBridgePromptPrefix,
} from "../../src/client-mcp/prompt.js";

const execTool = [{ name: "exec", inputFormat: "grammar" as const }];

describe("edgeOwnedBridgePromptPrefix", () => {
  it("describes MCP transport without GPT-6 persona or shell examples", () => {
    const complement = edgeOwnedBridgePromptPrefix([
      {
        name: "write",
        parameters: {
          type: "object",
          properties: { path: { type: "string" }, fileText: { type: "string" } },
        },
      },
    ]);

    expect(complement).toContain("CLIENT MCP TRANSPORT");
    expect(complement).toContain('providerIdentifier must be "client"');
    expect(complement).toContain("write: function_call JSON args");
    expect(complement).toContain("Schema fields: path, fileText");
    expect(complement).not.toContain("Composer model emulating GPT-6");
    expect(complement).not.toContain("git status -sb");
    expect(complement).not.toContain("GPT-6 OUTPUT CONTRACT");
    expect(complement).not.toContain("CODEX CLIENT INTERFACE");
  });

  it("prefers exec in the primary mcp example when Codex-shaped inventory includes function tools", () => {
    const complement = edgeOwnedBridgePromptPrefix([
      { name: "exec", inputFormat: "grammar" },
      {
        name: "client_shell",
        parameters: {
          type: "object",
          properties: { command: { type: "string" } },
          required: ["command"],
        },
      },
    ]);
    expect(complement).toMatch(/Call mcp with \{"providerIdentifier":"client","toolName":"exec"/);
    expect(complement).toContain("Function tools use a separate top-level mcp call");
    expect(complement).toContain('"toolName":"client_shell"');
    expect(complement).toContain(
      "Not inside exec: client_shell — use top-level mcp with toolName",
    );
    expect(complement).not.toContain(
      "Declared Codex nested tools: client_shell",
    );
  });

  it("labels exec as custom_tool_call transport without exec_command tutorial", () => {
    const complement = edgeOwnedBridgePromptPrefix(execTool);
    expect(complement).toContain("exec: custom_tool_call");
    expect(complement).toContain("EXEC ISOLATE");
    expect(complement).toContain("tools.exec_command");
    expect(complement).toContain("Never use require");
    expect(complement).toContain("tools.read");
    expect(complement).not.toContain("git status -sb");
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
      clientTools: execTool,
      workingDirectory: "/workspace/repo",
    });

    const rtkAt = text.indexOf("Always prefix shell commands with rtk");
    const complementAt = text.indexOf("CLIENT MCP TRANSPORT");
    const inputAt = text.indexOf("apply a one-line fix");
    const workdirAt = text.indexOf("IDE workspace");

    expect(rtkAt).toBeGreaterThanOrEqual(0);
    expect(complementAt).toBeGreaterThan(rtkAt);
    expect(inputAt).toBeGreaterThan(complementAt);
    expect(workdirAt).toBeGreaterThan(inputAt);
    expect(text).not.toContain("Composer model emulating GPT-6");
    expect(text).not.toContain("git status -sb");
  });

  it("omits instructions block when clientInstructions is missing", () => {
    const text = composeEdgeOwnedUserMessageText({
      prompt: "Ship it",
      clientTools: execTool,
    });
    expect(text.startsWith("CLIENT MCP TRANSPORT")).toBe(true);
    expect(text).toContain("Ship it");
  });
});
