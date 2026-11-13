import { describe, expect, it } from "vitest";
import { instructionsToPlainText } from "../../src/protocol/transcripts/helpers.js";
import { prepareResponsesPassthrough } from "../../src/protocol/prepare.js";

describe("instructionsToPlainText", () => {
  it("preserves string instructions", () => {
    expect(instructionsToPlainText("  alpha  ")).toBe("  alpha  ");
  });

  it("joins array instruction parts in order without duplication", () => {
    const normalized = instructionsToPlainText([
      "Always prefix shell commands with rtk",
      { type: "input_text", text: "You are Codex." },
    ]);
    expect(normalized).toBe("Always prefix shell commands with rtk\nYou are Codex.");
  });
});

describe("prepareResponsesPassthrough instructions split", () => {
  it("puts only serialized input in prompt.text while metadata keeps instructions", () => {
    const body = {
      model: "composer-2.5",
      instructions: [
        "Always prefix shell commands with rtk",
        { type: "input_text", text: "You are Codex, a coding agent." },
      ],
      input: [
        {
          type: "message",
          role: "user",
          content: [{ type: "input_text", text: "apply a one-line fix" }],
        },
      ],
    };
    const prepared = prepareResponsesPassthrough(body, { id: "composer-2.5" });
    expect(prepared.prompt.text).toContain("apply a one-line fix");
    expect(prepared.prompt.text).not.toContain("Always prefix shell commands with rtk");
    expect(prepared.prompt.text).not.toContain("You are Codex");
    expect(prepared.responseMetadata.instructions).toBe(
      "Always prefix shell commands with rtk\nYou are Codex, a coding agent.",
    );
    expect(prepared.passthroughBody?.instructions).toEqual(body.instructions);
  });
});
