import { describe, expect, it } from "vitest";
import { composeAntigravityUserMessage } from "../apps/api/src/utils/antigravity-prompt.js";

describe("composeAntigravityUserMessage", () => {
  it("places normalized instructions before the input on hub passthrough", () => {
    const text = composeAntigravityUserMessage({
      passthrough: true,
      instructions: ["You are Codex.", { type: "input_text", text: "Run git status before reviewing." }],
      toolsActive: true,
      inputText: "USER: review the local diff",
    });

    const instructionsAt = text.indexOf("You are Codex.");
    const skillAt = text.indexOf("Run git status before reviewing.");
    const inputAt = text.indexOf("USER: review the local diff");

    expect(instructionsAt).toBe(0);
    expect(skillAt).toBeGreaterThan(instructionsAt);
    expect(inputAt).toBeGreaterThan(skillAt);
    expect(text).toContain("MCP tool exec");
    expect(text).toContain("tools.exec_command");
    expect(text).toContain("one short sentence");
    expect(text).not.toContain("rtk ");
  });

  it("keeps the input alone when the request is not hub passthrough", () => {
    expect(
      composeAntigravityUserMessage({
        passthrough: false,
        instructions: "You are Codex.",
        toolsActive: true,
        inputText: "USER: hello",
      }),
    ).toBe("USER: hello");
  });

  it("omits the tool line when tools are not offered", () => {
    const text = composeAntigravityUserMessage({
      passthrough: true,
      instructions: "You are Codex.",
      toolsActive: false,
      inputText: "USER: hello",
    });
    expect(text).toBe("You are Codex.\n\nUSER: hello");
    expect(text).not.toContain("connexto_tools");
  });
});
