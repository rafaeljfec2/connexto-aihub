import { describe, expect, it } from "vitest";
import { prepareResponsesPassthrough, prepareResponsesRequest } from "../../src/protocol/prepare.js";
import { SYSTEM_DIRECTIVE } from "../../src/protocol/constants.js";

describe("prepareResponsesRequest direct-client mode", () => {
  it("keeps tool_choice none on the chat directive", () => {
    const prepared = prepareResponsesRequest(
      {
        model: "composer-2.5",
        tool_choice: "none",
        input: "explain a queue",
      },
      { id: "composer-2.5" },
    );

    expect(prepared.prompt.mode).toBe("ask");
    expect(prepared.prompt.text.startsWith(SYSTEM_DIRECTIVE)).toBe(true);
  });

  it("does not use passthrough on direct clients", () => {
    const prepared = prepareResponsesRequest(
      { model: "composer-2.5", input: "hello" },
      { id: "composer-2.5" },
    );
    expect(prepared.passthroughBody).toBeUndefined();
  });
});

describe("prepareResponsesPassthrough hub path", () => {
  it("never injects SYSTEM_DIRECTIVE for Codex-shaped bodies", () => {
    const prepared = prepareResponsesPassthrough(
      {
        model: "composer-2.5",
        instructions: "You are Codex",
        input: "fix tests",
        tools: [{ type: "custom", name: "apply_patch" }],
      },
      { id: "composer-2.5" },
    );
    expect(prepared.passthroughBody).toBeDefined();
    expect(prepared.prompt.text).not.toContain("SYSTEM_DIRECTIVE");
  });
});
