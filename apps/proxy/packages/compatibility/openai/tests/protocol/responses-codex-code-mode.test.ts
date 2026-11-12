import { describe, expect, it } from "vitest";
import { prepareResponsesRequest } from "../../src/protocol/prepare.js";
import {
  RESPONSES_TOOL_SYSTEM_DIRECTIVE,
  SYSTEM_DIRECTIVE,
} from "../../src/protocol/constants.js";

describe("prepareResponsesRequest Codex code mode without classic tools", () => {
  it("treats aihub/auto as an agent turn when tools are omitted", () => {
    const prepared = prepareResponsesRequest(
      {
        model: "aihub/auto",
        input: "fix the failing test",
      },
      { id: "composer-2.5" },
    );

    expect(prepared.prompt.mode).toBe("agent");
    expect(prepared.prompt.text.startsWith(RESPONSES_TOOL_SYSTEM_DIRECTIVE)).toBe(
      true,
    );
  });

  it("keeps tool_choice none on the chat directive", () => {
    const prepared = prepareResponsesRequest(
      {
        model: "aihub/auto",
        tool_choice: "none",
        input: "explain a queue",
      },
      { id: "composer-2.5" },
    );

    expect(prepared.prompt.mode).toBe("ask");
    expect(prepared.prompt.text.startsWith(SYSTEM_DIRECTIVE)).toBe(true);
  });

  it("keeps the agent directive after the gateway rewrites the model id", () => {
    const prepared = prepareResponsesRequest(
      {
        model: "composer-2.5",
        instructions: "You are Codex, a coding agent.",
        input: "fix the failing test",
      },
      { id: "composer-2.5" },
    );

    expect(prepared.prompt.mode).toBe("agent");
    expect(prepared.prompt.text.startsWith(RESPONSES_TOOL_SYSTEM_DIRECTIVE)).toBe(
      true,
    );
  });

  it("leaves a short chat turn on the ask directive", () => {
    const prepared = prepareResponsesRequest(
      {
        model: "composer-2.5",
        input: "explain a queue",
      },
      { id: "composer-2.5" },
    );

    expect(prepared.prompt.mode).toBe("ask");
    expect(prepared.prompt.text.startsWith(SYSTEM_DIRECTIVE)).toBe(true);
  });
});
