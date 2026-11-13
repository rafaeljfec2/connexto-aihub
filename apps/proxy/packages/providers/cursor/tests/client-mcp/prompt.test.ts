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
    expect(prompt).toContain("Use apply_patch for file changes.");
    expect(prompt).toContain("node:fs");
  });
});
