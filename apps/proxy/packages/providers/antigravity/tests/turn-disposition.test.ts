import { describe, expect, it } from "vitest";
import { acpTurnDisposition } from "../src/provider.js";
import { caughtTurnOutcome, delegatedToolTurnDecision } from "../src/turn-disposition.js";

describe("acpTurnDisposition", () => {
  it("completes the OpenAI turn when a client tool stopped Gemini", () => {
    expect(
      acpTurnDisposition({
        userCancelled: false,
        stopReason: "cancelled",
        toolCallCount: 1,
      }),
    ).toBe("completed");
  });

  it("keeps a user abort cancelled even if a tool was recorded", () => {
    expect(
      acpTurnDisposition({
        userCancelled: true,
        stopReason: "cancelled",
        toolCallCount: 1,
      }),
    ).toBe("cancelled");
  });
});

describe("delegated tool_choice decisions", () => {
  it("fails a specific tool_choice when a different tool was recorded", () => {
    expect(
      delegatedToolTurnDecision({
        userCancelled: false,
        toolChoice: { type: "function", function: { name: "required_tool" } },
        recordedToolNames: ["other_tool"],
      }),
    ).toEqual({
      status: "failed",
      reason: "Model failed to call required tool 'required_tool' despite tool_choice directive.",
    });
  });

  it("fails tool_choice required when nothing was recorded", () => {
    expect(
      delegatedToolTurnDecision({
        userCancelled: false,
        toolChoice: "required",
        recordedToolNames: [],
      }).status,
    ).toBe("failed");
  });

  it("completes when the named tool was recorded", () => {
    expect(
      delegatedToolTurnDecision({
        userCancelled: false,
        toolChoice: { type: "function", function: { name: "required_tool" } },
        recordedToolNames: ["required_tool"],
      }).status,
    ).toBe("completed");
  });

  it("stays cancelled when the user aborted", () => {
    expect(
      delegatedToolTurnDecision({
        userCancelled: true,
        toolChoice: "auto",
        recordedToolNames: ["other_tool"],
      }).status,
    ).toBe("cancelled");
  });
});

describe("caughtTurnOutcome", () => {
  it("does not treat a required-tool mismatch as a successful delegated stop", () => {
    expect(
      caughtTurnOutcome({
        userCancelled: false,
        intentionalDelegatedStop: true,
        toolChoice: { type: "function", function: { name: "required_tool" } },
        recordedToolNames: ["other_tool"],
      }),
    ).toBe("failed");
  });

  it("does not turn an unrelated prompt failure into success just because a tool was recorded", () => {
    expect(
      caughtTurnOutcome({
        userCancelled: false,
        intentionalDelegatedStop: false,
        toolChoice: "auto",
        recordedToolNames: ["other_tool"],
      }),
    ).toBe("failed");
  });

  it("completes the intentional delegated stop when tool_choice is satisfied", () => {
    expect(
      caughtTurnOutcome({
        userCancelled: false,
        intentionalDelegatedStop: true,
        toolChoice: undefined,
        recordedToolNames: ["other_tool"],
      }),
    ).toBe("completed");
  });

  it("keeps a user abort cancelled", () => {
    expect(
      caughtTurnOutcome({
        userCancelled: true,
        intentionalDelegatedStop: true,
        toolChoice: "auto",
        recordedToolNames: ["other_tool"],
      }),
    ).toBe("cancelled");
  });
});
