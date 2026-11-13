import { describe, expect, it } from "vitest";
import { prepareResponsesPassthrough, prepareResponsesRequest } from "../../src/protocol/prepare.js";
import {
  RESPONSES_TOOL_SYSTEM_DIRECTIVE,
  SYSTEM_DIRECTIVE,
} from "../../src/protocol/constants.js";

const toolProbeBody = {
  model: "composer-2.5",
  instructions: "You are Codex, a coding agent.",
  tool_choice: "auto",
  tools: [{ type: "custom", name: "apply_patch", description: "patch" }],
  input: [
    {
      type: "message",
      role: "user",
      content: [{ type: "input_text", text: "apply a one-line fix" }],
    },
  ],
};

describe("prepareResponsesPassthrough", () => {
  it("preserves instructions, tools, tool_choice, and input without directives", () => {
    const prepared = prepareResponsesPassthrough(toolProbeBody, { id: "composer-2.5" });

    expect(prepared.passthroughBody).toBeDefined();
    expect(prepared.prompt.text).not.toContain("SYSTEM_DIRECTIVE");
    expect(prepared.prompt.text).not.toContain(SYSTEM_DIRECTIVE);
    expect(prepared.prompt.text).not.toContain(RESPONSES_TOOL_SYSTEM_DIRECTIVE);
    expect(prepared.passthroughBody?.instructions).toBe(toolProbeBody.instructions);
    expect(prepared.passthroughBody?.tool_choice).toBe("auto");
    expect(prepared.passthroughBody?.tools).toEqual(toolProbeBody.tools);
    expect(prepared.passthroughBody?.input).toEqual(toolProbeBody.input);
    expect(prepared.prompt.text).toContain("apply a one-line fix");
    expect(prepared.prompt.text).not.toContain("You are Codex, a coding agent.");
  });

  it("does not inject agent directives on aihub-shaped Codex turns", () => {
    const prepared = prepareResponsesPassthrough(
      { ...toolProbeBody, model: "composer-2.5" },
      { id: "composer-2.5" },
    );
    const rewritten = prepareResponsesRequest(
      { model: "aihub/auto", input: "fix the failing test" },
      { id: "composer-2.5" },
    );

    expect(prepared.passthroughBody?.input).toEqual(toolProbeBody.input);
    expect(rewritten.prompt.text.startsWith(RESPONSES_TOOL_SYSTEM_DIRECTIVE)).toBe(
      false,
    );
  });
});
