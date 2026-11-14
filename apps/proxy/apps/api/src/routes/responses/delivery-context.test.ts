import { describe, expect, it } from "vitest";
import { extractActiveResponseToolResults } from "./delivery-context.js";

describe("extractActiveResponseToolResults", () => {
  it("reads custom_tool_call_output after a custom tool call", () => {
    const results = extractActiveResponseToolResults([
      { type: "custom_tool_call", call_id: "call_exec_1", name: "exec", input: "text(1);" },
      { type: "custom_tool_call_output", call_id: "call_exec_1", output: "TOKEN-OK" },
    ]);
    expect(results).toEqual([{ toolCallId: "call_exec_1", content: "TOKEN-OK" }]);
  });

  it("still reads function_call_output", () => {
    const results = extractActiveResponseToolResults([
      { type: "function_call", call_id: "call_1", name: "wait", arguments: "{}" },
      { type: "function_call_output", call_id: "call_1", output: "done" },
    ]);
    expect(results).toEqual([{ toolCallId: "call_1", content: "done" }]);
  });
});
