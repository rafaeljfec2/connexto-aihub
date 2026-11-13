import { describe, expect, it } from "vitest";
import { edgeOwnedBridgePromptPrefix } from "../../src/client-mcp/prompt.js";

describe("edgeOwnedBridgePromptPrefix", () => {
  it("describes function tools as function_call arguments", () => {
    const prompt = edgeOwnedBridgePromptPrefix("Create note.txt", [
      {
        name: "write",
        parameters: {
          type: "object",
          properties: { path: { type: "string" }, fileText: { type: "string" } },
        },
      },
    ]);

    expect(prompt).toContain("OPENAI OUTPUT CONTRACT");
    expect(prompt).toContain("write is a function_call");
    expect(prompt).toContain("Properties: path, fileText");
    expect(prompt).toContain('"toolName":"write"');
    expect(prompt).toContain('providerIdentifier "client"');
    expect(prompt).not.toContain("client_shell");
    expect(prompt.indexOf("GetDynamicTools")).toBeGreaterThan(prompt.indexOf("Create note.txt"));
  });

  it("teaches exec as a custom_tool_call with the Codex exec_command shape", () => {
    const prompt = edgeOwnedBridgePromptPrefix("Review the diff", [
      { name: "exec", inputFormat: "grammar", description: "run js" },
    ]);

    expect(prompt).toContain("exec is a custom_tool_call");
    expect(prompt).toContain("tools.exec_command");
    expect(prompt).toContain("custom_tool_call.input");
    expect(prompt).toContain("Do not pass a raw bash pipeline");
    expect(prompt).toContain("tools.client_write");
    expect(prompt).not.toContain("Prefer exact client tools");
  });

  it("points file edits at the function tool instead of exec", () => {
    const prompt = edgeOwnedBridgePromptPrefix("Update the runner", [
      { name: "exec", inputFormat: "grammar" },
      {
        name: "apply_patch",
        parameters: { type: "object", properties: { patch: { type: "string" } } },
      },
    ]);

    expect(prompt).toContain('"toolName":"apply_patch"');
    expect(prompt).toContain("Use apply_patch for file changes");
    expect(prompt).toContain("*** Begin Patch");
    expect(prompt).toContain("await tools.apply_patch(patchText)");
    expect(prompt).toContain("node:fs");
    expect(prompt).toContain("CODEX CLIENT INTERFACE");
    expect(prompt).toContain("tools.exec_command({ cmd:");
  });

  it("requires user-visible progress before mcp instead of silent tool calls", () => {
    const prompt = edgeOwnedBridgePromptPrefix("Corrigir o runner", [
      { name: "exec", inputFormat: "grammar" },
    ]);
    expect(prompt).toContain("USER VISIBLE PROGRESS (REQUIRED)");
    expect(prompt).toContain("Never run a client tool silently");
    expect(prompt).toContain("write the USER VISIBLE PROGRESS lines first");
    expect(prompt).not.toContain("Do not answer with prose before that mcp call");
    expect(prompt).not.toContain("No prose before mcp");
  });

  it("includes the mandatory GPT-6 contract block shared with the proxy validator", () => {
    const prompt = edgeOwnedBridgePromptPrefix("Ship the fix", [
      { name: "exec", inputFormat: "grammar" },
    ]);
    expect(prompt).toContain("GPT-6 OUTPUT CONTRACT (MANDATORY)");
    expect(prompt).toContain("The proxy validates every mcp payload");
    expect(prompt).toContain("Composer model emulating GPT-6");
  });
});
