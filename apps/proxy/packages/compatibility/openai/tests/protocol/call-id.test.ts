import { describe, expect, it } from "vitest";
import { responsesCallId } from "../../src/protocol/call-id.js";
import { responseToolCallEvents } from "../../src/protocol/sse.js";

describe("responses call_id length", () => {
  it("keeps ids that already fit the 64 character Responses limit", () => {
    expect(responsesCallId("call_exec_1")).toBe("call_exec_1");
  });

  it("maps a 106 character id to a stable id of at most 64 characters", () => {
    const longId = `call_${"a".repeat(101)}`;
    expect(longId).toHaveLength(106);
    const fitted = responsesCallId(longId);
    expect(fitted.length).toBeLessThanOrEqual(64);
    expect(fitted).toBe(responsesCallId(longId));
    expect(fitted.startsWith("call_")).toBe(true);

    const frames = Buffer.concat(
      responseToolCallEvents({
        id: "resp_test",
        outputIndex: 0,
        toolCall: {
          id: longId,
          type: "function",
          function: { name: "exec", arguments: "{}" },
        },
      }),
    ).toString("utf8");
    expect(frames).toContain(`"call_id":"${fitted}"`);
    expect(frames).not.toContain(longId);
  });
});
