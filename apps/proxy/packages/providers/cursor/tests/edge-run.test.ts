import { describe, expect, it } from "vitest";
import { agentRunFromCursorTextStream } from "../src/edge-run.js";

describe("agentRunFromCursorTextStream", () => {
  it("forwards the SDK call id on tool.call without synthesizing call_<name>_<seq>", async () => {
    const run = agentRunFromCursorTextStream({
      providerId: "cursor",
      sessionId: "sess-1",
      model: "composer-2.5",
      runId: "run-1",
      stream: (async function* () {
        yield {
          type: "tool_call",
          toolCall: {
            name: "write",
            arguments: { path: "a.txt" },
            sdkCallId: "tool_abc",
          },
        };
        yield { type: "done", finalText: "", toolCalls: [] };
      })(),
    });

    const events = [];
    for await (const event of run.events) events.push(event);
    expect(events).toContainEqual(
      expect.objectContaining({
        type: "tool.call",
        toolName: "write",
        callId: "tool_abc",
      }),
    );
    expect(events.some((event) => event.type === "tool.call" && event.callId === "call_write_1")).toBe(
      false,
    );
  });
});
